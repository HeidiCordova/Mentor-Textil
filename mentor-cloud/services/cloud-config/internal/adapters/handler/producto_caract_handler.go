package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-config/internal/adapters"
	"cloud-config/internal/ports"

	"mentor.local/shared/multitenancy"
)

// ─── Structs ──────────────────────────────────────────────────────────────────

type LineaProductoVar struct {
	ID         int    `json:"id"`
	LineaID    int    `json:"linea_id"`
	VariableID int    `json:"variable_id"`
	NombreCol  string `json:"nombre_col"`
	Orden      int    `json:"orden"`
	// extras para el frontend
	VariableNombre string `json:"variable_nombre,omitempty"`
	VariableClave  string `json:"variable_clave,omitempty"`
}

type ProductoCaract struct {
	ID         int       `json:"id"`
	ProductoID int       `json:"producto_id"`
	LineaID    int       `json:"linea_id"`
	VariableID int       `json:"variable_id"`
	Valor      string    `json:"valor"`
	CreadoEn   time.Time `json:"creado_en"`
}

// payload de entrada para guardar características
type CaractInput struct {
	LineaID   int                     `json:"linea_id"`
	Productos []ProductoCaractPayload `json:"productos"`
}

type ProductoCaractPayload struct {
	ProductoID int                  `json:"producto_id"`
	Valores    []ValorCaractPayload `json:"valores"`
}

type ValorCaractPayload struct {
	VariableID int    `json:"variable_id"`
	Valor      string `json:"valor"`
}

// payload para configurar columnas de una línea
type LineaVarsInput struct {
	LineaID  int                 `json:"linea_id"`
	Columnas []ColumnaVarPayload `json:"columnas"`
}

type ColumnaVarPayload struct {
	VariableID int    `json:"variable_id"`
	NombreCol  string `json:"nombre_col"`
	Orden      int    `json:"orden"`
}

// respuesta del GET de características
type CaractResponse struct {
	Columnas  []LineaProductoVar  `json:"columnas"`
	Productos []ProductoConCaract `json:"productos"`
}

type ProductoConCaract struct {
	ID        int               `json:"id"`
	Codigo    string            `json:"codigo"`
	Nombre    string            `json:"nombre"`
	EmpresaID *int              `json:"empresa_id,omitempty"`
	Activo    bool              `json:"activo"`
	Valores   map[int]ValorItem `json:"valores"` // variable_id → {id, valor}
}

type ValorItem struct {
	ID    int    `json:"id"`
	Valor string `json:"valor"`
}

// ─── Registro de Rutas ────────────────────────────────────────────────────────

func RegisterProductoCaractRoutes(r *gin.RouterGroup, masterDB *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	g := r.Group("", mw.Auth())

	// GET /productos?linea_id=X — devuelve el catálogo de productos de la línea
	g.GET("/productos", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rows, err := pool.Query(c.Request.Context(),
			fmt.Sprintf(`SELECT id, codigo, nombre, activo FROM %s.productos ORDER BY nombre`, schema))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type Producto struct {
			ID     int    `json:"id"`
			Codigo string `json:"codigo"`
			Nombre string `json:"nombre"`
			Activo bool   `json:"activo"`
		}
		var result []Producto
		for rows.Next() {
			var p Producto
			if err := rows.Scan(&p.ID, &p.Codigo, &p.Nombre, &p.Activo); err != nil {
				continue
			}
			result = append(result, p)
		}
		if result == nil {
			result = []Producto{}
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	// Variables definidas en la línea (fuente de verdad para el configurador de columnas)
	g.GET("/variables-linea", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		rows, err := pool.Query(c.Request.Context(),
			fmt.Sprintf(`SELECT id, nombre, tipo, activo FROM %s.variables WHERE activo = true ORDER BY id`, schema))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type varRow struct {
			ID     int    `json:"id"`
			Nombre string `json:"nombre"`
			Tipo   string `json:"tipo"`
			Activo bool   `json:"activo"`
		}
		result := []varRow{}
		for rows.Next() {
			var v varRow
			if err := rows.Scan(&v.ID, &v.Nombre, &v.Tipo, &v.Activo); err != nil {
				continue
			}
			result = append(result, v)
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	// Columnas (variables) configuradas para una línea
	g.GET("/linea-producto-vars", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, lineaID)
		if err != nil {
			return
		}
		cols, err := getLineaProductoVarsPlanta(c.Request.Context(), pool, schema, lineaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": cols})
	})

	g.PUT("/linea-producto-vars", func(c *gin.Context) {
		var body LineaVarsInput
		if err := c.ShouldBindJSON(&body); err != nil || body.LineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, body.LineaID)
		if err != nil {
			return
		}
		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())

		if _, err = tx.Exec(c.Request.Context(),
			fmt.Sprintf(`DELETE FROM %s.linea_producto_vars`, schema)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, col := range body.Columnas {
			if _, err = tx.Exec(c.Request.Context(),
				fmt.Sprintf(`INSERT INTO %s.linea_producto_vars (variable_id, nombre_col, orden)
				 VALUES ($1,$2,$3)
				 ON CONFLICT (variable_id) DO UPDATE SET nombre_col=EXCLUDED.nombre_col, orden=EXCLUDED.orden`, schema),
				col.VariableID, col.NombreCol, col.Orden); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		// Limpiar producto_caracteristicas huérfanas (variables eliminadas de la config)
		if len(body.Columnas) > 0 {
			placeholders := make([]string, len(body.Columnas))
			args := make([]interface{}, len(body.Columnas))
			for i, col := range body.Columnas {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
				args[i] = col.VariableID
			}
			deleteOrphansSQL := fmt.Sprintf(
				`DELETE FROM %s.producto_caracteristicas WHERE variable_id NOT IN (%s)`,
				schema, strings.Join(placeholders, ","),
			)
			if _, err = tx.Exec(c.Request.Context(), deleteOrphansSQL, args...); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			// Sin columnas configuradas → limpiar todas las características
			if _, err = tx.Exec(c.Request.Context(),
				fmt.Sprintf(`DELETE FROM %s.producto_caracteristicas`, schema)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if err = tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, int64(body.LineaID)); eid > 0 {
				go notifier.Broadcast(context.Background(), "linea_producto_vars", eid, nil)
			}
		}
		cols, _ := getLineaProductoVarsPlanta(c.Request.Context(), pool, schema, body.LineaID)
		c.JSON(http.StatusOK, gin.H{"data": cols})
	})

	// Características de productos por línea
	g.GET("/producto-caracteristicas", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, lineaID)
		if err != nil {
			return
		}

		// columnas configuradas
		cols, err := getLineaProductoVarsPlanta(c.Request.Context(), pool, schema, lineaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// todos los productos activos de la línea (directamente en su schema)
		rows, err := pool.Query(c.Request.Context(),
			fmt.Sprintf(`SELECT id, codigo, nombre, activo FROM %s.productos WHERE activo = true ORDER BY codigo`, schema))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var productos []ProductoConCaract
		for rows.Next() {
			var pr ProductoConCaract
			if err := rows.Scan(&pr.ID, &pr.Codigo, &pr.Nombre, &pr.Activo); err != nil {
				continue
			}
			pr.Valores = make(map[int]ValorItem)
			productos = append(productos, pr)
		}
		rows.Close()

		if len(productos) > 0 {
			idMap := make(map[int]int) // producto_id → index en slice
			for i, p := range productos {
				idMap[p.ID] = i
			}

			vrows, err := pool.Query(c.Request.Context(),
				fmt.Sprintf(`SELECT id, producto_id, variable_id, valor FROM %s.producto_caracteristicas`, schema))
			if err == nil {
				defer vrows.Close()
				for vrows.Next() {
					var id, pid, vid int
					var valor string
					if err := vrows.Scan(&id, &pid, &vid, &valor); err != nil {
						continue
					}
					if idx, ok := idMap[pid]; ok {
						productos[idx].Valores[vid] = ValorItem{ID: id, Valor: valor}
					}
				}
			}
		}

		if productos == nil {
			productos = []ProductoConCaract{}
		}
		c.JSON(http.StatusOK, CaractResponse{Columnas: cols, Productos: productos})
	})

	g.PUT("/producto-caracteristicas", func(c *gin.Context) {
		var body CaractInput
		if err := c.ShouldBindJSON(&body); err != nil || body.LineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, body.LineaID)
		if err != nil {
			return
		}

		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())

		for _, prod := range body.Productos {
			// auto-crear registro de velocidad nominal (velocidad=0, pendiente configurar)
			if _, err := tx.Exec(c.Request.Context(),
				fmt.Sprintf(`INSERT INTO %s.velocidad_nominal (producto_id, velocidad_us, factor_conv)
				VALUES ($1,0,1) ON CONFLICT (producto_id) DO NOTHING`, schema),
				prod.ProductoID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// Borrar todas las características actuales del producto (enfoque clean-slate)
			// Evita duplicados y limpia valores huérfanos de variables eliminadas
			if _, err := tx.Exec(c.Request.Context(),
				fmt.Sprintf(`DELETE FROM %s.producto_caracteristicas WHERE producto_id = $1`, schema),
				prod.ProductoID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// Insertar solo valores no vacíos
			for _, v := range prod.Valores {
				if strings.TrimSpace(v.Valor) == "" {
					continue
				}
				if _, err := tx.Exec(c.Request.Context(),
					fmt.Sprintf(`INSERT INTO %s.producto_caracteristicas (producto_id, variable_id, valor)
					VALUES ($1,$2,$3)`, schema),
					prod.ProductoID, v.VariableID, v.Valor); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}

		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, int64(body.LineaID)); eid > 0 {
				go notifier.Broadcast(context.Background(), "producto_caracteristicas", eid, nil)
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// ─── Catálogo de valores permitidos por variable ──────────────────────

	// GET /linea-var-catalogo?linea_id=X&variable_id=Y
	g.GET("/linea-var-catalogo", func(c *gin.Context) {
		lineaID, err1 := strconv.Atoi(c.Query("linea_id"))
		variableID, err2 := strconv.Atoi(c.Query("variable_id"))
		if err1 != nil || lineaID == 0 || err2 != nil || variableID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id y variable_id requeridos"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, lineaID)
		if err != nil {
			return
		}
		valores, err := getCatalogoPlanta(c.Request.Context(), pool, schema, variableID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": valores})
	})

	// GET /linea-var-catalogo/all?linea_id=X  → TODO el catálogo de la línea agrupado por variable_id
	g.GET("/linea-var-catalogo/all", func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, lineaID)
		if err != nil {
			return
		}
		rows, err := pool.Query(c.Request.Context(),
			fmt.Sprintf(`SELECT variable_id, valor, orden FROM %s.linea_var_catalogo ORDER BY variable_id, orden, id`, schema))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		result := map[int][]string{}
		for rows.Next() {
			var vid, orden int
			var valor string
			if err := rows.Scan(&vid, &valor, &orden); err != nil {
				continue
			}
			result[vid] = append(result[vid], valor)
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	// PUT /linea-var-catalogo  body: {linea_id, variable_id, valores: ["A","B",...]}
	g.PUT("/linea-var-catalogo", func(c *gin.Context) {
		var body struct {
			LineaID    int      `json:"linea_id"`
			VariableID int      `json:"variable_id"`
			Valores    []string `json:"valores"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.LineaID == 0 || body.VariableID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id, variable_id y valores requeridos"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, body.LineaID)
		if err != nil {
			return
		}
		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())
		// reemplazar catálogo completo para esta variable
		if _, err = tx.Exec(c.Request.Context(),
			fmt.Sprintf(`DELETE FROM %s.linea_var_catalogo WHERE variable_id=$1`, schema),
			body.VariableID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for i, v := range body.Valores {
			if v == "" {
				continue
			}
			if _, err = tx.Exec(c.Request.Context(),
				fmt.Sprintf(`INSERT INTO %s.linea_var_catalogo (variable_id, valor, orden)
				 VALUES ($1,$2,$3) ON CONFLICT (variable_id, valor) DO UPDATE SET orden=EXCLUDED.orden`, schema),
				body.VariableID, v, i); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if err = tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		valores, _ := getCatalogoPlanta(c.Request.Context(), pool, schema, body.VariableID)
		c.JSON(http.StatusOK, gin.H{"data": valores})
	})

	// POST /productos-crear — crea el producto directamente en el schema de la línea
	g.POST("/productos-crear", func(c *gin.Context) {
		var body struct {
			Codigo  string `json:"codigo"`
			Nombre  string `json:"nombre"`
			LineaID int    `json:"linea_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Codigo == "" || body.LineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "codigo, nombre y linea_id requeridos"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, body.LineaID)
		if err != nil {
			return
		}
		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())
		var id int
		err = tx.QueryRow(c.Request.Context(),
			fmt.Sprintf(`INSERT INTO %s.productos (codigo, nombre, activo) VALUES ($1,$2,true) RETURNING id`, schema),
			body.Codigo, body.Nombre).Scan(&id)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "SKU ya existe o error al crear: " + err.Error()})
			return
		}
		// auto-crear registro de velocidad nominal
		if _, err := tx.Exec(c.Request.Context(),
			fmt.Sprintf(`INSERT INTO %s.velocidad_nominal (producto_id, velocidad_us, factor_conv) VALUES ($1,0,1) ON CONFLICT DO NOTHING`, schema),
			id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, int64(body.LineaID)); eid > 0 {
				go notifier.Broadcast(context.Background(), "productos", eid, nil)
				go notifier.Broadcast(context.Background(), "velocidad_nominal", eid, nil)
			}
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	})

	// DELETE /productos-crear/:id?linea_id=X — desactiva el producto en el schema de la línea
	g.DELETE("/productos-crear/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		pool, schema, err := resolvePlantaSchema(c, masterDB, mgr, lineaID)
		if err != nil {
			return
		}
		if _, err := pool.Exec(c.Request.Context(),
			fmt.Sprintf(`UPDATE %s.productos SET activo=false WHERE id=$1`, schema), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if notifier != nil {
			if eid := resolveEmpresaIDWithFallback(c.Request.Context(), c, masterDB, int64(lineaID)); eid > 0 {
				go notifier.Broadcast(context.Background(), "productos", eid, nil)
				go notifier.Broadcast(context.Background(), "velocidad_nominal", eid, nil)
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

// ─── Helpers de planta ────────────────────────────────────────────────────────

func getCatalogoPlanta(ctx context.Context, pool *pgxpool.Pool, schema string, variableID int) ([]string, error) {
	rows, err := pool.Query(ctx,
		fmt.Sprintf(`SELECT valor FROM %s.linea_var_catalogo WHERE variable_id=$1 ORDER BY orden, id`, schema),
		variableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			continue
		}
		result = append(result, v)
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}

func getLineaProductoVarsPlanta(ctx context.Context, pool *pgxpool.Pool, schema string, lineaID int) ([]LineaProductoVar, error) {
	rows, err := pool.Query(ctx,
		fmt.Sprintf(`SELECT lpv.id, lpv.variable_id, lpv.nombre_col, lpv.orden, v.nombre
		FROM %s.linea_producto_vars lpv
		JOIN %s.variables v ON v.id = lpv.variable_id
		ORDER BY lpv.orden, lpv.id`, schema, schema))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LineaProductoVar
	for rows.Next() {
		var lpv LineaProductoVar
		lpv.LineaID = lineaID
		if err := rows.Scan(&lpv.ID, &lpv.VariableID, &lpv.NombreCol, &lpv.Orden, &lpv.VariableNombre); err != nil {
			continue
		}
		lpv.VariableClave = lpv.VariableNombre // sin columna clave; usamos nombre como fallback
		result = append(result, lpv)
	}
	if result == nil {
		result = []LineaProductoVar{}
	}
	return result, nil
}
