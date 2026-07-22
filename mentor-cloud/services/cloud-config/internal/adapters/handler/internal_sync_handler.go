package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"mentor.local/shared/multitenancy"
)

// ── Internal sync structs (match edge SYNC payload format) ───────────────────

type scopedCatItem struct {
	ID         int64  `json:"id"`
	Nombre     string `json:"nombre"`
	Codigo     string `json:"codigo"`
	PadreID    *int64 `json:"padre_id,omitempty"`
	Orden      int    `json:"orden"`
	Activo     bool   `json:"activo"`
	TipoParada string `json:"tipo_parada,omitempty"`
	EmpresaID  *int   `json:"empresa_id,omitempty"`
	LineaID    *int   `json:"linea_id,omitempty"`
}

type scopedProducto struct {
	ID          int     `json:"id"`
	Codigo      string  `json:"codigo"`
	Nombre      string  `json:"nombre"`
	EmpresaID   *int    `json:"empresa_id,omitempty"`
	Activo      bool    `json:"activo"`
	LineaID     *int    `json:"linea_id,omitempty"`
	VelocidadUS float64 `json:"velocidad_us"`
	FactorConv  int     `json:"factor_conv"`
}

type scopedTurno struct {
	ID         int    `json:"id"`
	Nombre     string `json:"nombre"`
	HoraInicio string `json:"hora_inicio"`
	HoraFin    string `json:"hora_fin"`
	PlantaID   *int   `json:"planta_id,omitempty"`
	Activo     bool   `json:"activo"`
}

type scopedVariable struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	Clave         string `json:"clave"`
	Valor         string `json:"valor"`
	Tipo          string `json:"tipo"`
	DispositivoID *int   `json:"dispositivo_id,omitempty"`
	PlantaID      *int   `json:"planta_id,omitempty"`
	EmpresaID     int    `json:"empresa_id"`
	Activo        bool   `json:"activo"`
}

type scopedLineaProductoVar struct {
	ID         int    `json:"id"`
	LineaID    int    `json:"linea_id"`
	VariableID int    `json:"variable_id"`
	NombreCol  string `json:"nombre_col"`
	Orden      int    `json:"orden"`
}

type scopedProductoCaract struct {
	ID         int    `json:"id"`
	ProductoID int    `json:"producto_id"`
	LineaID    int    `json:"linea_id"`
	VariableID int    `json:"variable_id"`
	Valor      string `json:"valor"`
}

type scopedUsuario struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Nombre       string `json:"nombre"`
	Apellido     string `json:"apellido"`
	PasswordHash string `json:"password_hash"`
	RolID        int    `json:"rol_id"`
	Rol          string `json:"rol"`
	EmpresaID    int    `json:"empresa_id"`
	PlantaID     *int   `json:"planta_id,omitempty"`
	Activo       bool   `json:"activo"`
}

type scopedPlanta struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	EmpresaID     int    `json:"empresa_id"`
	EmpresaNombre string `json:"empresa_nombre"`
	Activo        bool   `json:"activo"`
}

type scopedVelocidadNominal struct {
	LineaID     int     `json:"linea_id"`
	ProductoID  int     `json:"producto_id"`
	VelocidadUs float64 `json:"velocidad_us"`
	FactorConv  int     `json:"factor_conv"`
}

type scopedMotivoVelocidad struct {
	ID     int    `json:"id"`
	Texto  string `json:"texto"`
	Activo bool   `json:"activo"`
	Orden  int    `json:"orden"`
}

type scopedLinea struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	PlantaID int    `json:"planta_id"`
	Tipo     string `json:"tipo"`
	Subtipo  string `json:"subtipo"`
	Activo   bool   `json:"activo"`
}

// ── Registration ─────────────────────────────────────────────────────────────

func RegisterInternalSyncRoutes(r *gin.Engine, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, internalKey string) {
	g := r.Group("/internal", internalKeyGuard(internalKey))
	g.POST("/device-config", deviceConfigHandler(db, mgr))
}

func internalKeyGuard(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Internal-Key") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

// ── Handler ──────────────────────────────────────────────────────────────────

func deviceConfigHandler(db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			EmpresaID int64 `json:"empresa_id"`
			PlantaID  int64 `json:"planta_id"`
			LineaID   int64 `json:"linea_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.EmpresaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "empresa_id requerido"})
			return
		}

		ctx := c.Request.Context()
		result := make(map[string]interface{})

		// Merge cat_programada + cat_no_programada en un solo array con tipo_parada
		// para que el gateway pueda despacharlo como SYNC_CATALOG al edge.
		var categorias []scopedCatItem
		if prog, err := fetchCatItems(ctx, db, mgr, "cat_programada", "PROGRAMADA", req.EmpresaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] cat_programada error: %v", err)
		} else {
			categorias = append(categorias, prog...)
		}
		if noProg, err := fetchCatItems(ctx, db, mgr, "cat_no_programada", "NO_PROGRAMADA", req.EmpresaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] cat_no_programada error: %v", err)
		} else {
			categorias = append(categorias, noProg...)
		}
		if categorias == nil {
			categorias = []scopedCatItem{}
		}
		result["categoria_paradas"] = categorias

		if prods, err := fetchProductos(ctx, db, mgr, req.EmpresaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] productos error: %v", err)
		} else {
			result["productos"] = prods
		}

		if turnos, err := fetchTurnos(ctx, db, mgr, req.EmpresaID, req.PlantaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] turnos error: %v", err)
		} else {
			result["turnos"] = turnos
		}

		if vars, err := fetchVariables(ctx, db, req.EmpresaID); err != nil {
			log.Printf("[internal-sync] variables error: %v", err)
		} else {
			result["variables"] = vars
		}

		if lpvs, err := fetchLineaProductoVars(ctx, db, mgr, req.EmpresaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] linea_producto_vars error: %v", err)
		} else {
			result["linea_producto_vars"] = lpvs
		}

		if pcs, err := fetchProductoCaracts(ctx, db, mgr, req.EmpresaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] producto_caracts error: %v", err)
		} else {
			result["producto_caracteristicas"] = pcs
		}

		if users, err := fetchUsuarios(ctx, db, req.EmpresaID, req.PlantaID); err != nil {
			log.Printf("[internal-sync] usuarios error: %v", err)
		} else {
			result["usuarios"] = users
		}

		if plData, err := fetchPlantasLineas(ctx, db, req.EmpresaID); err != nil {
			log.Printf("[internal-sync] plantas_lineas error: %v", err)
		} else {
			result["plantas_lineas"] = plData
		}

		if vnData, err := fetchVelocidadNominal(ctx, db, mgr, req.EmpresaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] velocidad_nominal error: %v", err)
		} else {
			result["velocidad_nominal"] = vnData
		}

		if mvData, err := fetchMotivosVelocidad(ctx, db, mgr, req.EmpresaID, req.LineaID); err != nil {
			log.Printf("[internal-sync] motivos_velocidad error: %v", err)
		} else {
			result["motivos_velocidad"] = mvData
		}

		c.JSON(http.StatusOK, result)
	}
}

// ── Query helpers ────────────────────────────────────────────────────────────

// empresaLineas devuelve los IDs de todas las líneas activas de una empresa.
func empresaLineas(ctx context.Context, db *pgxpool.Pool, empresaID int64) ([]int64, error) {
	rows, err := db.Query(ctx,
		`SELECT l.id FROM config.lineas l
		 JOIN config.plantas p ON p.id = l.planta_id
		 WHERE p.empresa_id = $1 AND l.activo = true
		 ORDER BY l.id`, empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// fetchFromPlanta es el helper genérico para cualquier tabla que vive en la BD de planta.
// Siempre usa resolvePlantaPool — imposible leer del lugar equivocado.
// Para agregar una tabla nueva: implementa un fetchXxx que llame a fetchFromPlanta.
func fetchFromPlanta[T any](
	ctx context.Context,
	db *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	empresaID, lineaID int64,
	scanFn func(pool *pgxpool.Pool, schema string, lid int64) ([]T, error),
) ([]T, error) {
	var lineas []int64
	if lineaID > 0 {
		lineas = []int64{lineaID}
	} else {
		var err error
		lineas, err = empresaLineas(ctx, db, empresaID)
		if err != nil {
			return nil, err
		}
	}
	var result []T
	for _, lid := range lineas {
		pool, schema, err := resolvePlantaPool(ctx, db, mgr, lid)
		if err != nil {
			return nil, fmt.Errorf("planta pool linea %d: %w", lid, err)
		}
		items, err := scanFn(pool, schema, lid)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}
	if result == nil {
		result = []T{}
	}
	return result, nil
}

func fetchCatItems(ctx context.Context, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, tabla, tipoParada string, empresaID, lineaID int64) ([]scopedCatItem, error) {
	return fetchFromPlanta(ctx, db, mgr, empresaID, lineaID, func(pool *pgxpool.Pool, schema string, lid int64) ([]scopedCatItem, error) {
		q := fmt.Sprintf(`SELECT id, nombre, codigo, padre_id, orden, activo FROM %s.%s WHERE activo = TRUE ORDER BY orden, id`, schema, tabla)
		rows, err := pool.Query(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("cat linea %d tabla %s: %w", lid, tabla, err)
		}
		defer rows.Close()
		lidInt := int(lid)
		eidInt := int(empresaID)
		var result []scopedCatItem
		for rows.Next() {
			var item scopedCatItem
			if err := rows.Scan(&item.ID, &item.Nombre, &item.Codigo, &item.PadreID, &item.Orden, &item.Activo); err != nil {
				return nil, err
			}
			item.TipoParada = tipoParada
			item.LineaID = &lidInt
			item.EmpresaID = &eidInt
			result = append(result, item)
		}
		return result, nil
	})
}

func fetchProductos(ctx context.Context, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, empresaID, lineaID int64) ([]scopedProducto, error) {
	return fetchFromPlanta(ctx, db, mgr, empresaID, lineaID, func(pool *pgxpool.Pool, schema string, lid int64) ([]scopedProducto, error) {
		query := fmt.Sprintf(`
			SELECT p.id, p.codigo, p.nombre, p.activo,
			       COALESCE(vn.velocidad_us,0), COALESCE(vn.factor_conv,1)
			FROM %s.productos p
			LEFT JOIN %s.velocidad_nominal vn ON vn.producto_id = p.id
			WHERE p.activo = true
			ORDER BY p.codigo`, schema, schema)
		rows, err := pool.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("productos linea %d: %w", lid, err)
		}
		defer rows.Close()
		lidInt := int(lid)
		var result []scopedProducto
		for rows.Next() {
			var p scopedProducto
			p.LineaID = &lidInt
			if err := rows.Scan(&p.ID, &p.Codigo, &p.Nombre, &p.Activo, &p.VelocidadUS, &p.FactorConv); err != nil {
				return nil, err
			}
			result = append(result, p)
		}
		return result, nil
	})
}

func fetchTurnos(ctx context.Context, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, empresaID, plantaID, lineaID int64) ([]scopedTurno, error) {
	var plantas []int64
	if plantaID > 0 {
		plantas = []int64{plantaID}
	} else {
		pRows, err := db.Query(ctx,
			`SELECT id FROM config.plantas WHERE empresa_id = $1 AND activo = true ORDER BY id`, empresaID)
		if err != nil {
			return nil, err
		}
		defer pRows.Close()
		for pRows.Next() {
			var pid int64
			if err := pRows.Scan(&pid); err != nil {
				return nil, err
			}
			plantas = append(plantas, pid)
		}
	}

	var result []scopedTurno
	for _, pid := range plantas {
		pool, err := mgr.Get(ctx, int(pid))
		if err != nil {
			return nil, fmt.Errorf("fetchTurnos planta %d: %w", pid, err)
		}
		pidInt := int(pid)

		// Primero intentar turnos específicos de la línea (linea_id = lineaID)
		if lineaID > 0 {
			lineaRows, err := pool.Query(ctx, `
				SELECT DISTINCT ON (nombre, hora_inicio, hora_fin)
					id, nombre, hora_inicio::TEXT, hora_fin::TEXT, activo
				FROM public.turno_dia
				WHERE linea_id = $1
				  AND vigente_desde = (
				    SELECT MAX(vigente_desde) FROM public.turno_dia
				    WHERE linea_id = $1
				  )
				  AND activo = true
				ORDER BY nombre, hora_inicio, hora_fin, id`, lineaID)
			if err != nil {
				return nil, fmt.Errorf("fetchTurnos linea %d: %w", lineaID, err)
			}
			var lineaTurnos []scopedTurno
			for lineaRows.Next() {
				var t scopedTurno
				t.PlantaID = &pidInt
				if err := lineaRows.Scan(&t.ID, &t.Nombre, &t.HoraInicio, &t.HoraFin, &t.Activo); err != nil {
					lineaRows.Close()
					return nil, err
				}
				lineaTurnos = append(lineaTurnos, t)
			}
			lineaRows.Close()
			if len(lineaTurnos) > 0 {
				log.Printf("[fetchTurnos] planta=%d linea=%d → %d turnos específicos de línea", pid, lineaID, len(lineaTurnos))
				result = append(result, lineaTurnos...)
				continue // turnos de línea encontrados, no usar fallback de planta
			}
			log.Printf("[fetchTurnos] planta=%d linea=%d → sin turnos de línea, usando fallback de planta", pid, lineaID)
		}

		// Fallback: turnos a nivel de planta (linea_id IS NULL)
		rows, err := pool.Query(ctx, `
			SELECT DISTINCT ON (nombre, hora_inicio, hora_fin)
				id, nombre, hora_inicio::TEXT, hora_fin::TEXT, activo
			FROM public.turno_dia
			WHERE linea_id IS NULL
			  AND vigente_desde = (
			    SELECT MAX(vigente_desde) FROM public.turno_dia
			    WHERE linea_id IS NULL
			  )
			  AND activo = true
			ORDER BY nombre, hora_inicio, hora_fin, id`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var t scopedTurno
			t.PlantaID = &pidInt
			if err := rows.Scan(&t.ID, &t.Nombre, &t.HoraInicio, &t.HoraFin, &t.Activo); err != nil {
				rows.Close()
				return nil, err
			}
			result = append(result, t)
		}
		rows.Close()
	}
	if result == nil {
		result = []scopedTurno{}
	}
	return result, nil
}

func fetchVariables(ctx context.Context, db *pgxpool.Pool, empresaID int64) ([]scopedVariable, error) {
	rows, err := db.Query(ctx,
		`SELECT id, nombre, clave, COALESCE(valor,''), tipo,
		        dispositivo_id, planta_id, COALESCE(empresa_id,0), activo
		 FROM config.variables
		 WHERE empresa_id = $1
		 ORDER BY id`, empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []scopedVariable
	for rows.Next() {
		var v scopedVariable
		if err := rows.Scan(&v.ID, &v.Nombre, &v.Clave, &v.Valor, &v.Tipo,
			&v.DispositivoID, &v.PlantaID, &v.EmpresaID, &v.Activo); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	if result == nil {
		result = []scopedVariable{}
	}
	return result, nil
}

func fetchLineaProductoVars(ctx context.Context, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, empresaID, lineaID int64) ([]scopedLineaProductoVar, error) {
	return fetchFromPlanta(ctx, db, mgr, empresaID, lineaID, func(pool *pgxpool.Pool, schema string, lid int64) ([]scopedLineaProductoVar, error) {
		query := fmt.Sprintf(`
			SELECT id, variable_id, nombre_col, orden
			FROM %s.linea_producto_vars
			ORDER BY orden, id`, schema)
		rows, err := pool.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("linea_producto_vars linea %d: %w", lid, err)
		}
		defer rows.Close()
		var result []scopedLineaProductoVar
		for rows.Next() {
			var lpv scopedLineaProductoVar
			lpv.LineaID = int(lid)
			if err := rows.Scan(&lpv.ID, &lpv.VariableID, &lpv.NombreCol, &lpv.Orden); err != nil {
				return nil, err
			}
			result = append(result, lpv)
		}
		return result, nil
	})
}

func fetchUsuarios(ctx context.Context, db *pgxpool.Pool, empresaID, plantaID int64) ([]scopedUsuario, error) {
	// Filtra: usuarios sin planta asignada (globales de empresa) +
	// usuarios asignados exactamente a esta planta.
	// Si plantaID==0 se devuelven todos los de la empresa (sin filtro de planta).
	var rows pgx.Rows
	var err error
	if plantaID > 0 {
		rows, err = db.Query(ctx,
			`SELECT u.id, u.username, COALESCE(u.email,''), u.nombre,
			        COALESCE(u.apellido,''), COALESCE(u.password_hash,''),
			        COALESCE(u.rol_id, 0), COALESCE(r.nombre, ''),
			        COALESCE(u.empresa_id, 0), u.planta_id, u.activo
			 FROM identity.usuarios u
			 LEFT JOIN identity.roles r ON r.id = u.rol_id
			 WHERE (u.empresa_id = $1 OR u.empresa_id IS NULL)
			   AND (u.planta_id IS NULL OR u.planta_id = $2)
			   AND u.activo = true
			 ORDER BY u.id`, empresaID, plantaID)
	} else {
		rows, err = db.Query(ctx,
			`SELECT u.id, u.username, COALESCE(u.email,''), u.nombre,
			        COALESCE(u.apellido,''), COALESCE(u.password_hash,''),
			        COALESCE(u.rol_id, 0), COALESCE(r.nombre, ''),
			        COALESCE(u.empresa_id, 0), u.planta_id, u.activo
			 FROM identity.usuarios u
			 LEFT JOIN identity.roles r ON r.id = u.rol_id
			 WHERE (u.empresa_id = $1 OR u.empresa_id IS NULL) AND u.activo = true
			 ORDER BY u.id`, empresaID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []scopedUsuario
	for rows.Next() {
		var u scopedUsuario
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Nombre,
			&u.Apellido, &u.PasswordHash,
			&u.RolID, &u.Rol, &u.EmpresaID, &u.PlantaID, &u.Activo); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	if result == nil {
		result = []scopedUsuario{}
	}
	return result, nil
}

type plantasLineasBundle struct {
	Plantas []scopedPlanta `json:"plantas"`
	Lineas  []scopedLinea  `json:"lineas"`
}

func fetchPlantasLineas(ctx context.Context, db *pgxpool.Pool, empresaID int64) (plantasLineasBundle, error) {
	var bundle plantasLineasBundle

	pRows, err := db.Query(ctx,
		`SELECT p.id, p.nombre, p.empresa_id, COALESCE(e.nombre,''), p.activo
		 FROM config.plantas p
		 LEFT JOIN identity.empresas e ON e.id = p.empresa_id
		 WHERE p.empresa_id = $1 AND p.activo = true ORDER BY p.id`, empresaID)
	if err != nil {
		return bundle, err
	}
	defer pRows.Close()
	for pRows.Next() {
		var p scopedPlanta
		if err := pRows.Scan(&p.ID, &p.Nombre, &p.EmpresaID, &p.EmpresaNombre, &p.Activo); err != nil {
			return bundle, err
		}
		bundle.Plantas = append(bundle.Plantas, p)
	}
	if bundle.Plantas == nil {
		bundle.Plantas = []scopedPlanta{}
	}

	lRows, err := db.Query(ctx,
		`SELECT l.id, l.nombre, l.planta_id,
		        COALESCE(l.tipo,''), COALESCE(l.subtipo,''), l.activo
		 FROM config.lineas l
		 JOIN config.plantas p ON p.id = l.planta_id
		 WHERE p.empresa_id = $1 AND l.activo = true ORDER BY l.id`, empresaID)
	if err != nil {
		return bundle, err
	}
	defer lRows.Close()
	for lRows.Next() {
		var l scopedLinea
		if err := lRows.Scan(&l.ID, &l.Nombre, &l.PlantaID, &l.Tipo, &l.Subtipo, &l.Activo); err != nil {
			return bundle, err
		}
		bundle.Lineas = append(bundle.Lineas, l)
	}
	if bundle.Lineas == nil {
		bundle.Lineas = []scopedLinea{}
	}
	return bundle, nil
}

func fetchProductoCaracts(ctx context.Context, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, empresaID, lineaID int64) ([]scopedProductoCaract, error) {
	return fetchFromPlanta(ctx, db, mgr, empresaID, lineaID, func(pool *pgxpool.Pool, schema string, lid int64) ([]scopedProductoCaract, error) {
		query := fmt.Sprintf(`
			SELECT id, producto_id, variable_id, COALESCE(valor,'')
			FROM %s.producto_caracteristicas
			ORDER BY producto_id, variable_id`, schema)
		rows, err := pool.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("producto_caracteristicas linea %d: %w", lid, err)
		}
		defer rows.Close()
		var result []scopedProductoCaract
		for rows.Next() {
			var pc scopedProductoCaract
			pc.LineaID = int(lid)
			if err := rows.Scan(&pc.ID, &pc.ProductoID, &pc.VariableID, &pc.Valor); err != nil {
				return nil, err
			}
			result = append(result, pc)
		}
		return result, nil
	})
}

func fetchMotivosVelocidad(ctx context.Context, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, empresaID, lineaID int64) ([]scopedMotivoVelocidad, error) {
	return fetchFromPlanta(ctx, db, mgr, empresaID, lineaID, func(pool *pgxpool.Pool, schema string, lid int64) ([]scopedMotivoVelocidad, error) {
		query := fmt.Sprintf(`
			SELECT id, texto, activo, orden
			FROM %s.motivos_velocidad
			WHERE activo = true
			ORDER BY orden, id`, schema)
		rows, err := pool.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("motivos_velocidad linea %d: %w", lid, err)
		}
		defer rows.Close()
		var result []scopedMotivoVelocidad
		for rows.Next() {
			var m scopedMotivoVelocidad
			if err := rows.Scan(&m.ID, &m.Texto, &m.Activo, &m.Orden); err != nil {
				return nil, err
			}
			result = append(result, m)
		}
		return result, nil
	})
}

func fetchVelocidadNominal(ctx context.Context, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, empresaID, lineaID int64) ([]scopedVelocidadNominal, error) {
	return fetchFromPlanta(ctx, db, mgr, empresaID, lineaID, func(pool *pgxpool.Pool, schema string, lid int64) ([]scopedVelocidadNominal, error) {
		query := fmt.Sprintf(`
			SELECT producto_id, velocidad_us, factor_conv
			FROM %s.velocidad_nominal
			ORDER BY producto_id`, schema)
		rows, err := pool.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("velocidad_nominal linea %d: %w", lid, err)
		}
		defer rows.Close()
		var result []scopedVelocidadNominal
		for rows.Next() {
			var v scopedVelocidadNominal
			v.LineaID = int(lid)
			if err := rows.Scan(&v.ProductoID, &v.VelocidadUs, &v.FactorConv); err != nil {
				return nil, err
			}
			result = append(result, v)
		}
		return result, nil
	})
}
