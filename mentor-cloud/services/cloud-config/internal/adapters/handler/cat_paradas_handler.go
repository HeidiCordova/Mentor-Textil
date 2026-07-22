package handler

// cat_paradas_handler.go
// Maneja los árboles de categorías de paradas v2.
// Las tablas cat_programada y cat_no_programada viven en
// {schema}.* dentro de cada BD de planta (mentor_planta_X).

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	"cloud-config/internal/adapters"
	"cloud-config/internal/ports"

	"mentor.local/shared/multitenancy"
)

const maxCatNiveles = 7

var nonAlphaNumCat = regexp.MustCompile(`[^A-Z0-9]+`)

func slugifyCat(parts []string) string {
	segs := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.ToUpper(strings.TrimSpace(p))
		s = nonAlphaNumCat.ReplaceAllString(s, "_")
		s = strings.Trim(s, "_")
		if s != "" {
			segs = append(segs, s)
		}
	}
	return strings.Join(segs, "-")
}

type catNodo struct {
	Codigo  string
	Nombre  string
	Orden   int
	PadreID *int64
}

// ──────────────────────────────────────────────────────────────
// Registro de rutas
// ──────────────────────────────────────────────────────────────

func RegisterCatParadasRoutes(
	r *gin.RouterGroup,
	masterDB *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	mw *ports.JWTMiddleware,
	notifier *adapters.SyncNotifier,
) {
	g := r.Group("/cat-paradas", mw.Auth())

	g.GET("/programadas", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		tree, err := getCatTreePlanta(c.Request.Context(), masterDB, mgr, "cat_programada", lineaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"linea_id": lineaID, "arbol": tree})
	})

	g.GET("/no-programadas", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		tree, err := getCatTreePlanta(c.Request.Context(), masterDB, mgr, "cat_no_programada", lineaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"linea_id": lineaID, "arbol": tree})
	})

	g.POST("/programadas/import", func(c *gin.Context) {
		lineaID, _ := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		importCatCSVPlanta(c, masterDB, mgr, "cat_programada")
		if notifier != nil && c.Writer.Status() < 300 {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, lineaID); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, lineaID)
			}
		}
	})

	g.POST("/no-programadas/import", func(c *gin.Context) {
		lineaID, _ := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		importCatCSVPlanta(c, masterDB, mgr, "cat_no_programada")
		if notifier != nil && c.Writer.Status() < 300 {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, lineaID); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, lineaID)
			}
		}
	})

	g.DELETE("/programadas/:id", func(c *gin.Context) {
		lineaID, _ := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		desactivarCatPlanta(c, masterDB, mgr, "cat_programada")
		if notifier != nil && c.Writer.Status() < 300 {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, lineaID); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, lineaID)
			}
		}
	})
	g.DELETE("/no-programadas/:id", func(c *gin.Context) {
		lineaID, _ := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		desactivarCatPlanta(c, masterDB, mgr, "cat_no_programada")
		if notifier != nil && c.Writer.Status() < 300 {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, lineaID); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, lineaID)
			}
		}
	})
}

// ──────────────────────────────────────────────────────────────
// GET árbol
// ──────────────────────────────────────────────────────────────

type catNodoResp struct {
	ID      int64         `json:"id"`
	Codigo  string        `json:"codigo"`
	Nombre  string        `json:"nombre"`
	Orden   int           `json:"orden"`
	Activo  bool          `json:"activo"`
	PadreID *int64        `json:"padre_id,omitempty"`
	Hijos   []catNodoResp `json:"hijos,omitempty"`
}

func getCatTreePlanta(ctx context.Context, masterDB *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, tabla string, lineaID int) ([]catNodoResp, error) {
	pool, schema, err := resolvePlantaPool(ctx, masterDB, mgr, int64(lineaID))
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf(`
SELECT id, codigo, nombre, orden, activo, padre_id
FROM %s.%s
WHERE activo = TRUE
ORDER BY orden, nombre`, schema, tabla)

	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[int64]*catNodoResp{}
	var roots []*catNodoResp

	for rows.Next() {
		var n catNodoResp
		if err := rows.Scan(&n.ID, &n.Codigo, &n.Nombre, &n.Orden, &n.Activo, &n.PadreID); err != nil {
			return nil, err
		}
		byID[n.ID] = &n
	}

	for _, n := range byID {
		if n.PadreID == nil {
			roots = append(roots, n)
		} else if padre, ok := byID[*n.PadreID]; ok {
			padre.Hijos = append(padre.Hijos, *n)
		}
	}

	result := make([]catNodoResp, 0, len(roots))
	for _, r := range roots {
		result = append(result, *r)
	}
	return result, nil
}

// ──────────────────────────────────────────────────────────────
// POST importar CSV/XLSX
// ──────────────────────────────────────────────────────────────

func importCatCSVPlanta(c *gin.Context, masterDB *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, tabla string) {
	lineaID, err := strconv.Atoi(c.Query("linea_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
		return
	}

	pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	file, header, err := c.Request.FormFile("archivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'archivo' requerido (CSV o XLSX)"})
		return
	}
	defer file.Close()

	var nodos []catNodo
	var parseErr error

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == ".xlsx" || ext == ".xls" {
		data, readErr := io.ReadAll(file)
		if readErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "leer archivo: " + readErr.Error()})
			return
		}
		xlFile, openErr := excelize.OpenReader(bytes.NewReader(data))
		if openErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "abrir XLSX: " + openErr.Error()})
			return
		}
		defer xlFile.Close()
		nodos, parseErr = parseCatXLSX(xlFile)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parsear XLSX: " + parseErr.Error()})
			return
		}
	} else {
		nodos, parseErr = parseCatCSV(file)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parsear CSV: " + parseErr.Error()})
			return
		}
	}

	if len(nodos) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "archivo sin filas validas"})
		return
	}

	tx, err := pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "iniciar tx: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())

	insertados := 0
	actualizados := 0
	codigoToID := map[string]int64{}

	existSQL := fmt.Sprintf(`SELECT id, codigo FROM %s.%s`, schema, tabla)
	eRows, _ := tx.Query(c.Request.Context(), existSQL)
	for eRows.Next() {
		var eid int64
		var ecod string
		_ = eRows.Scan(&eid, &ecod)
		codigoToID[ecod] = eid
	}
	eRows.Close()

	for _, n := range nodos {
		var padreIDVal *int64
		if n.PadreID != nil {
			parts := strings.Split(n.Codigo, "-")
			if len(parts) > 1 {
				padreCode := strings.Join(parts[:len(parts)-1], "-")
				if pid, ok := codigoToID[padreCode]; ok {
					padreIDVal = &pid
				}
			}
		}

		if existID, exists := codigoToID[n.Codigo]; exists {
			_, err = tx.Exec(c.Request.Context(),
				fmt.Sprintf(`UPDATE %s.%s SET nombre=$1, orden=$2, activo=TRUE WHERE id=$3`, schema, tabla),
				n.Nombre, n.Orden, existID)
			if err != nil {
				tx.Rollback(c.Request.Context())
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("update %s: %v", n.Codigo, err)})
				return
			}
			actualizados++
		} else {
			var newID int64
			err = tx.QueryRow(c.Request.Context(),
				fmt.Sprintf(`INSERT INTO %s.%s (codigo, nombre, padre_id, orden, activo) VALUES ($1, $2, $3, $4, TRUE) RETURNING id`, schema, tabla),
				n.Codigo, n.Nombre, padreIDVal, n.Orden,
			).Scan(&newID)
			if err != nil {
				tx.Rollback(c.Request.Context())
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("insert %s: %v", n.Codigo, err)})
				return
			}
			codigoToID[n.Codigo] = newID
			insertados++
		}
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "commit: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tabla":        tabla,
		"linea_id":     lineaID,
		"insertados":   insertados,
		"actualizados": actualizados,
		"total":        len(nodos),
	})
}

// parseCatXLSX lee un archivo Excel (.xlsx).
func parseCatXLSX(f *excelize.File) ([]catNodo, error) {
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("el archivo no contiene hojas")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("leer hoja '%s': %w", sheets[0], err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("hoja sin filas de datos (solo cabecera o vacia)")
	}

	seen := map[string]bool{}
	var result []catNodo

	for _, record := range rows[1:] {
		niveles := make([]string, 0, maxCatNiveles)
		for i := 0; i < maxCatNiveles && i < len(record); i++ {
			v := strings.TrimSpace(record[i])
			if v == "" {
				break
			}
			niveles = append(niveles, v)
		}
		if len(niveles) == 0 {
			continue
		}

		orden := 0
		if len(record) > maxCatNiveles {
			orden, _ = strconv.Atoi(strings.TrimSpace(record[maxCatNiveles]))
		}

		for depth := 1; depth <= len(niveles); depth++ {
			path := niveles[:depth]
			codigo := slugifyCat(path)
			if seen[codigo] {
				continue
			}
			seen[codigo] = true

			var padreIDPtr *int64
			if depth > 1 {
				placeholder := int64(-1)
				padreIDPtr = &placeholder
			}

			nodeOrden := 0
			if depth == len(niveles) {
				nodeOrden = orden
			}

			result = append(result, catNodo{
				Codigo:  codigo,
				Nombre:  path[depth-1],
				Orden:   nodeOrden,
				PadreID: padreIDPtr,
			})
		}
	}

	return result, nil
}

// parseCatCSV lee el CSV y devuelve los nodos en orden de inserción.
func parseCatCSV(r io.Reader) ([]catNodo, error) {
	reader := csv.NewReader(r)
	reader.Comma = ','
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("leer cabecera: %w", err)
	}

	seen := map[string]bool{}
	var result []catNodo

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		niveles := make([]string, 0, maxCatNiveles)
		for i := 0; i < maxCatNiveles && i < len(record); i++ {
			v := strings.TrimSpace(record[i])
			if v == "" {
				break
			}
			niveles = append(niveles, v)
		}
		if len(niveles) == 0 {
			continue
		}

		orden := 0
		if len(record) > maxCatNiveles {
			orden, _ = strconv.Atoi(strings.TrimSpace(record[maxCatNiveles]))
		}

		for depth := 1; depth <= len(niveles); depth++ {
			path := niveles[:depth]
			codigo := slugifyCat(path)
			if seen[codigo] {
				continue
			}
			seen[codigo] = true

			var padreIDPtr *int64
			if depth > 1 {
				placeholder := int64(-1)
				padreIDPtr = &placeholder
			}

			nodeOrden := 0
			if depth == len(niveles) {
				nodeOrden = orden
			}

			result = append(result, catNodo{
				Codigo:  codigo,
				Nombre:  path[depth-1],
				Orden:   nodeOrden,
				PadreID: padreIDPtr,
			})
		}
	}

	return result, nil
}

// ──────────────────────────────────────────────────────────────
// DELETE (desactivar) — requiere ?linea_id=X
// ──────────────────────────────────────────────────────────────

func desactivarCatPlanta(c *gin.Context, masterDB *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, tabla string) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	lineaID, err := strconv.Atoi(c.Query("linea_id"))
	if err != nil || lineaID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
		return
	}
	pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	_, err = pool.Exec(c.Request.Context(),
		fmt.Sprintf(`UPDATE %s.%s SET activo = FALSE WHERE id = $1`, schema, tabla), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"desactivado": id})
}

// ──────────────────────────────────────────────────────────────
// Fila plana (para árbol y exportar)
// ──────────────────────────────────────────────────────────────

type catFlatRow struct {
	ID                int64  `json:"id"`
	Categoria         string `json:"categoria"`
	Subcategoria      string `json:"subcategoria"`
	Subcategoria2     string `json:"subcategoria_2"`
	Subcategoria3     string `json:"subcategoria_3"`
	Subcategoria4     string `json:"subcategoria_4"`
	DescripcionParada string `json:"descripcion_parada"`
	Maquina           string `json:"maquina"`
}

// flatLeafRows devuelve los nodos hoja con su ruta de hasta 7 niveles.
func flatLeafRows(ctx context.Context, pool *pgxpool.Pool, schema, tabla string) ([]catFlatRow, error) {
	q := fmt.Sprintf(`
SELECT
  leaf.id,
  COALESCE(n1.nombre, n2.nombre, n3.nombre, n4.nombre, n5.nombre, n6.nombre, leaf.nombre) AS cat,
  CASE
    WHEN n1.id IS NOT NULL THEN COALESCE(n2.nombre, n3.nombre, n4.nombre, n5.nombre, n6.nombre, leaf.nombre)
    WHEN n2.id IS NOT NULL THEN COALESCE(n3.nombre, n4.nombre, n5.nombre, n6.nombre, leaf.nombre)
    WHEN n3.id IS NOT NULL THEN COALESCE(n4.nombre, n5.nombre, n6.nombre, leaf.nombre)
    WHEN n4.id IS NOT NULL THEN COALESCE(n5.nombre, n6.nombre, leaf.nombre)
    WHEN n5.id IS NOT NULL THEN COALESCE(n6.nombre, leaf.nombre)
    WHEN n6.id IS NOT NULL THEN leaf.nombre
    ELSE ''
  END AS sub,
  CASE
    WHEN n1.id IS NOT NULL THEN COALESCE(n3.nombre, n4.nombre, n5.nombre, n6.nombre, leaf.nombre)
    WHEN n2.id IS NOT NULL THEN COALESCE(n4.nombre, n5.nombre, n6.nombre, leaf.nombre)
    WHEN n3.id IS NOT NULL THEN COALESCE(n5.nombre, n6.nombre, leaf.nombre)
    WHEN n4.id IS NOT NULL THEN COALESCE(n6.nombre, leaf.nombre)
    WHEN n5.id IS NOT NULL THEN leaf.nombre
    ELSE ''
  END AS sub2,
  CASE
    WHEN n1.id IS NOT NULL THEN COALESCE(n4.nombre, n5.nombre, n6.nombre, leaf.nombre)
    WHEN n2.id IS NOT NULL THEN COALESCE(n5.nombre, n6.nombre, leaf.nombre)
    WHEN n3.id IS NOT NULL THEN COALESCE(n6.nombre, leaf.nombre)
    WHEN n4.id IS NOT NULL THEN leaf.nombre
    ELSE ''
  END AS sub3,
  CASE
    WHEN n1.id IS NOT NULL THEN COALESCE(n5.nombre, n6.nombre, leaf.nombre)
    WHEN n2.id IS NOT NULL THEN COALESCE(n6.nombre, leaf.nombre)
    WHEN n3.id IS NOT NULL THEN leaf.nombre
    ELSE ''
  END AS sub4,
  CASE
    WHEN n1.id IS NOT NULL THEN COALESCE(n6.nombre, leaf.nombre)
    WHEN n2.id IS NOT NULL THEN leaf.nombre
    ELSE ''
  END AS desc_parada,
  CASE
    WHEN n1.id IS NOT NULL THEN leaf.nombre
    ELSE ''
  END AS maquina
FROM %s.%s leaf
LEFT JOIN %s.%s n6 ON n6.id = leaf.padre_id
LEFT JOIN %s.%s n5 ON n5.id = n6.padre_id
LEFT JOIN %s.%s n4 ON n4.id = n5.padre_id
LEFT JOIN %s.%s n3 ON n3.id = n4.padre_id
LEFT JOIN %s.%s n2 ON n2.id = n3.padre_id
LEFT JOIN %s.%s n1 ON n1.id = n2.padre_id
WHERE leaf.activo = TRUE
  AND leaf.id NOT IN (
    SELECT DISTINCT padre_id FROM %s.%s WHERE padre_id IS NOT NULL
  )
ORDER BY cat, sub, sub2, sub3, sub4, desc_parada, maquina`,
		schema, tabla,
		schema, tabla,
		schema, tabla,
		schema, tabla,
		schema, tabla,
		schema, tabla,
		schema, tabla,
		schema, tabla)

	rows, err := pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []catFlatRow
	for rows.Next() {
		var r catFlatRow
		if err := rows.Scan(&r.ID, &r.Categoria, &r.Subcategoria, &r.Subcategoria2, &r.Subcategoria3, &r.Subcategoria4, &r.DescripcionParada, &r.Maquina); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	if result == nil {
		result = []catFlatRow{}
	}
	return result, nil
}

// flatRowsToNodos convierte filas planas (hasta 7 niveles) al mismo formato
// que producen parseCatCSV/parseCatXLSX.
func flatRowsToNodos(rows []catFlatRow) []catNodo {
	seen := map[string]bool{}
	var result []catNodo

	for _, r := range rows {
		if r.Categoria == "" {
			continue
		}
		niveles := []string{r.Categoria}
		if r.Subcategoria != "" {
			niveles = append(niveles, r.Subcategoria)
		}
		if r.Subcategoria2 != "" {
			niveles = append(niveles, r.Subcategoria2)
		}
		if r.Subcategoria3 != "" {
			niveles = append(niveles, r.Subcategoria3)
		}
		if r.Subcategoria4 != "" {
			niveles = append(niveles, r.Subcategoria4)
		}
		if r.DescripcionParada != "" {
			niveles = append(niveles, r.DescripcionParada)
		}
		if r.Maquina != "" {
			niveles = append(niveles, r.Maquina)
		}

		for depth := 1; depth <= len(niveles); depth++ {
			path := niveles[:depth]
			codigo := slugifyCat(path)
			if seen[codigo] {
				continue
			}
			seen[codigo] = true
			var padreIDPtr *int64
			if depth > 1 {
				placeholder := int64(-1)
				padreIDPtr = &placeholder
			}
			result = append(result, catNodo{
				Codigo:  codigo,
				Nombre:  path[depth-1],
				Orden:   0,
				PadreID: padreIDPtr,
			})
		}
	}
	return result
}

// upsertNodos realiza insert/update de una lista de catNodo en una tabla planta.
// Los nodos que estaban activos en la DB pero ya NO están en la nueva lista se
// marcan como activo=FALSE (soft-delete).
// Devuelve (insertados, actualizados, eliminados, error).
func upsertNodos(ctx context.Context, pool *pgxpool.Pool, schema, tabla string, nodos []catNodo) (int, int, int, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback(ctx)

	type existingNode struct {
		id     int64
		activo bool
	}
	codigoToNode := map[string]existingNode{}
	eRows, eErr := tx.Query(ctx, fmt.Sprintf(`SELECT id, codigo, activo FROM %s.%s`, schema, tabla))
	if eErr != nil {
		return 0, 0, 0, fmt.Errorf("read existing %s: %w", tabla, eErr)
	}
	for eRows.Next() {
		var eid int64
		var ecod string
		var eactivo bool
		_ = eRows.Scan(&eid, &ecod, &eactivo)
		codigoToNode[ecod] = existingNode{id: eid, activo: eactivo}
	}
	eRows.Close()

	// Mapa auxiliar id→codigo para resolver padre_id correctamente
	codigoToID := map[string]int64{}
	for cod, n := range codigoToNode {
		codigoToID[cod] = n.id
	}

	// Conjunto de códigos presentes en el nuevo import
	incomingCodes := map[string]bool{}
	for _, n := range nodos {
		incomingCodes[n.Codigo] = true
	}

	ins, upd, del := 0, 0, 0
	for _, n := range nodos {
		var padreIDVal *int64
		if n.PadreID != nil {
			parts := strings.Split(n.Codigo, "-")
			if len(parts) > 1 {
				padreCode := strings.Join(parts[:len(parts)-1], "-")
				if pid, ok := codigoToID[padreCode]; ok {
					padreIDVal = &pid
				}
			}
		}
		if existing, exists := codigoToNode[n.Codigo]; exists {
			_, err = tx.Exec(ctx,
				fmt.Sprintf(`UPDATE %s.%s SET nombre=$1, orden=$2, activo=TRUE WHERE id=$3`, schema, tabla),
				n.Nombre, n.Orden, existing.id)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("update %s: %w", n.Codigo, err)
			}
			upd++
		} else {
			var newID int64
			err = tx.QueryRow(ctx,
				fmt.Sprintf(`INSERT INTO %s.%s (codigo, nombre, padre_id, orden, activo) VALUES ($1,$2,$3,$4,TRUE) RETURNING id`, schema, tabla),
				n.Codigo, n.Nombre, padreIDVal, n.Orden,
			).Scan(&newID)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("insert %s: %w", n.Codigo, err)
			}
			codigoToID[n.Codigo] = newID
			ins++
		}
	}

	// Desactivar nodos que estaban activos pero ya no están en el import
	for cod, existing := range codigoToNode {
		if existing.activo && !incomingCodes[cod] {
			_, err = tx.Exec(ctx,
				fmt.Sprintf(`UPDATE %s.%s SET activo=FALSE WHERE id=$1`, schema, tabla),
				existing.id)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("deactivate %s: %w", cod, err)
			}
			del++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, fmt.Errorf("commit: %w", err)
	}
	return ins, upd, del, nil
}

// ──────────────────────────────────────────────────────────────
// /arbol-paradas  —  árbol consolidado (GET, POST importar, GET exportar)
// ──────────────────────────────────────────────────────────────

func RegisterArbolParadasRoutes(
	r *gin.RouterGroup,
	masterDB *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	mw *ports.JWTMiddleware,
	notifier *adapters.SyncNotifier,
) {
	g := r.Group("/arbol-paradas", mw.Auth())

	// GET /arbol-paradas?linea_id=X → { programadas:[...], no_programadas:[...] }
	g.GET("", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		prog, err := flatLeafRows(c.Request.Context(), pool, schema, "cat_programada")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		noProg, err := flatLeafRows(c.Request.Context(), pool, schema, "cat_no_programada")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"programadas": prog, "no_programadas": noProg})
	})

	// GET /arbol-paradas/exportar?linea_id=X → same format as GET
	g.GET("/exportar", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		prog, err := flatLeafRows(c.Request.Context(), pool, schema, "cat_programada")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		noProg, err := flatLeafRows(c.Request.Context(), pool, schema, "cat_no_programada")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"programadas": prog, "no_programadas": noProg})
	})

	// POST /arbol-paradas/importar
	// Body: { linea_id, programadas: [...catFlatRow], no_programadas: [...catFlatRow] }
	g.POST("/importar", func(c *gin.Context) {
		var body struct {
			LineaID       int64        `json:"linea_id"`
			Programadas   []catFlatRow `json:"programadas"`
			NoProgramadas []catFlatRow `json:"no_programadas"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.LineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id y datos requeridos"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, body.LineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		insP, updP, delP, err := upsertNodos(c.Request.Context(), pool, schema, "cat_programada", flatRowsToNodos(body.Programadas))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "programadas: " + err.Error()})
			return
		}
		insNP, updNP, delNP, err := upsertNodos(c.Request.Context(), pool, schema, "cat_no_programada", flatRowsToNodos(body.NoProgramadas))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no_programadas: " + err.Error()})
			return
		}

		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, body.LineaID); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, body.LineaID)
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"inserted": insP + insNP,
			"updated":  updP + updNP,
			"diff": gin.H{
				"agregadas":   insP + insNP,
				"modificadas": updP + updNP,
				"eliminadas":  delP + delNP,
			},
		})
	})
}

// ──────────────────────────────────────────────────────────────
// /categoria-paradas  —  CRUD plano de categorías por linea
// ──────────────────────────────────────────────────────────────

type catItemResp struct {
	ID      int64  `json:"id"`
	Codigo  string `json:"codigo"`
	Nombre  string `json:"nombre"`
	Orden   int    `json:"orden"`
	Activo  bool   `json:"activo"`
	PadreID *int64 `json:"padre_id,omitempty"`
	Tipo    string `json:"tipo"` // "programada" | "no_programada"
}

func RegisterCategoriaParadasRoutes(
	r *gin.RouterGroup,
	masterDB *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	mw *ports.JWTMiddleware,
	notifier *adapters.SyncNotifier,
) {
	g := r.Group("/categoria-paradas", mw.Auth())

	tipoTabla := func(tipo string) string {
		if tipo == "no_programada" {
			return "cat_no_programada"
		}
		return "cat_programada"
	}

	// GET /categoria-paradas?linea_id=X[&tipo=programada|no_programada]
	g.GET("", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		tipo := c.DefaultQuery("tipo", "")
		var result []catItemResp

		queryTabla := func(tabla, tipoLabel string) error {
			q := fmt.Sprintf(`SELECT id, codigo, nombre, orden, activo, padre_id FROM %s.%s WHERE activo=TRUE ORDER BY orden, nombre`, schema, tabla)
			rows, err := pool.Query(c.Request.Context(), q)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var item catItemResp
				item.Tipo = tipoLabel
				if err := rows.Scan(&item.ID, &item.Codigo, &item.Nombre, &item.Orden, &item.Activo, &item.PadreID); err != nil {
					return err
				}
				result = append(result, item)
			}
			return nil
		}

		if tipo == "no_programada" {
			if err := queryTabla("cat_no_programada", "no_programada"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else if tipo == "programada" {
			if err := queryTabla("cat_programada", "programada"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			if err := queryTabla("cat_programada", "programada"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if err := queryTabla("cat_no_programada", "no_programada"); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if result == nil {
			result = []catItemResp{}
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	// POST /categoria-paradas?linea_id=X&tipo=programada|no_programada
	g.POST("", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		tabla := tipoTabla(c.DefaultQuery("tipo", "programada"))

		var body struct {
			Nombre  string `json:"nombre"`
			Codigo  string `json:"codigo"`
			PadreID *int64 `json:"padre_id"`
			Orden   int    `json:"orden"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Nombre == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nombre requerido"})
			return
		}
		if body.Codigo == "" {
			body.Codigo = slugifyCat([]string{body.Nombre})
		}

		var newID int64
		err = pool.QueryRow(c.Request.Context(),
			fmt.Sprintf(`INSERT INTO %s.%s (codigo, nombre, padre_id, orden, activo) VALUES ($1,$2,$3,$4,TRUE) RETURNING id`, schema, tabla),
			body.Codigo, body.Nombre, body.PadreID, body.Orden,
		).Scan(&newID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, int64(lineaID)); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, int64(lineaID))
			}
		}
		c.JSON(http.StatusCreated, gin.H{"id": newID, "codigo": body.Codigo, "nombre": body.Nombre})
	})

	// PUT /categoria-paradas/:id?linea_id=X&tipo=programada|no_programada
	g.PUT("/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
			return
		}
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		tabla := tipoTabla(c.DefaultQuery("tipo", "programada"))

		var body struct {
			Nombre string `json:"nombre"`
			Orden  int    `json:"orden"`
			Activo *bool  `json:"activo"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		activo := true
		if body.Activo != nil {
			activo = *body.Activo
		}
		_, err = pool.Exec(c.Request.Context(),
			fmt.Sprintf(`UPDATE %s.%s SET nombre=$1, orden=$2, activo=$3 WHERE id=$4`, schema, tabla),
			body.Nombre, body.Orden, activo, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, int64(lineaID)); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, int64(lineaID))
			}
		}
		c.JSON(http.StatusOK, gin.H{"updated": id})
	})

	// DELETE /categoria-paradas/:id?linea_id=X&tipo=programada|no_programada
	g.DELETE("/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
			return
		}
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		tabla := tipoTabla(c.DefaultQuery("tipo", "programada"))
		_, err = pool.Exec(c.Request.Context(),
			fmt.Sprintf(`UPDATE %s.%s SET activo=FALSE WHERE id=$1`, schema, tabla), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, int64(lineaID)); eid > 0 {
				go notifier.Broadcast(context.Background(), "categoria_paradas", eid, nil)
				go notifier.NotifyBrowsers(context.Background(), "catalog.changed", eid, int64(lineaID))
			}
		}
		c.JSON(http.StatusOK, gin.H{"desactivado": id})
	})
}
