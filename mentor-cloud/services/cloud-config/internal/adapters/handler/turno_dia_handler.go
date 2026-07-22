package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"cloud-config/internal/adapters"
	"cloud-config/internal/ports"

	"mentor.local/shared/multitenancy"
)

type TurnoDia struct {
	ID                int       `json:"id"`
	PlantaID          int       `json:"planta_id"`
	LineaID           *int      `json:"linea_id"`
	DiaSemana         int       `json:"dia_semana"`
	Nombre            string    `json:"nombre"`
	HoraInicio        string    `json:"hora_inicio"`
	HoraFin           string    `json:"hora_fin"`
	Color             string    `json:"color"`
	Activo            bool      `json:"activo"`
	RenovacionSemanal bool      `json:"renovacion_semanal"`
	VigenteDesdé      string    `json:"vigente_desde"`
	CreadoEn          time.Time `json:"creado_en"`
}

type VersionInfo struct {
	VigenteDesdé string `json:"vigente_desde"`
	Total        int    `json:"total_turnos"`
}

func RegisterTurnoDiaRoutes(r *gin.RouterGroup, masterDB *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	g := r.Group("/turno-dias", mw.Auth())

	// GET /turno-dias?planta_id=X[&linea_id=Y][&fecha=YYYY-MM-DD]
	g.GET("", func(c *gin.Context) {
		plantaID, err := strconv.Atoi(c.Query("planta_id"))
		if err != nil || plantaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "planta_id requerido"})
			return
		}

		var lineaID *int
		if raw := c.Query("linea_id"); raw != "" {
			v, err := strconv.Atoi(raw)
			if err == nil {
				lineaID = &v
			}
		}

		fechaRef := c.Query("fecha")
		if fechaRef == "" {
			fechaRef = time.Now().Format("2006-01-02")
		}

		pool, err := mgr.Get(c.Request.Context(), plantaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()

		var vigenteDesde string
		var versionQuery string
		var versionArgs []any

		if lineaID != nil {
			versionQuery = `
SELECT vigente_desde::TEXT FROM public.turno_dia
WHERE linea_id = $1 AND vigente_desde <= $2::DATE
ORDER BY vigente_desde DESC LIMIT 1`
			versionArgs = []any{*lineaID, fechaRef}
		} else {
			versionQuery = `
SELECT vigente_desde::TEXT FROM public.turno_dia
WHERE linea_id IS NULL AND vigente_desde <= $1::DATE
ORDER BY vigente_desde DESC LIMIT 1`
			versionArgs = []any{fechaRef}
		}

		// Buscar historial completo siempre, independiente de si hay versión vigente
		var histQuery string
		var histArgs []any
		if lineaID != nil {
			histQuery = `SELECT vigente_desde::TEXT, COUNT(*)::INT FROM public.turno_dia WHERE linea_id = $1 GROUP BY vigente_desde ORDER BY vigente_desde DESC`
			histArgs = []any{*lineaID}
		} else {
			histQuery = `SELECT vigente_desde::TEXT, COUNT(*)::INT FROM public.turno_dia WHERE linea_id IS NULL GROUP BY vigente_desde ORDER BY vigente_desde DESC`
			histArgs = []any{}
		}

		hRows, err := pool.Query(ctx, histQuery, histArgs...)
		var versiones []VersionInfo
		if err == nil {
			defer hRows.Close()
			for hRows.Next() {
				var vigente string
				var total int64
				if err := hRows.Scan(&vigente, &total); err != nil {
					log.Printf("turno-dias hist scan error planta=%d linea=%v err=%v", plantaID, lineaID, err)
					continue
				}
				versiones = append(versiones, VersionInfo{VigenteDesdé: vigente, Total: int(total)})
			}
		} else {
			log.Printf("turno-dias hist query error planta=%d linea=%v err=%v", plantaID, lineaID, err)
		}
		if versiones == nil {
			versiones = []VersionInfo{}
		}
		log.Printf("turno-dias historial planta=%d linea=%v fecha_ref=%s versiones=%d", plantaID, lineaID, fechaRef, len(versiones))

		row := pool.QueryRow(ctx, versionQuery, versionArgs...)
		if err := row.Scan(&vigenteDesde); err != nil {
			log.Printf("turno-dias sin vigente planta=%d linea=%v fecha_ref=%s", plantaID, lineaID, fechaRef)
			c.JSON(http.StatusOK, gin.H{
				"data":               []TurnoDia{},
				"vigente_desde":      "",
				"renovacion_semanal": true,
				"versiones":          versiones,
			})
			return
		}

		var dataQuery string
		var dataArgs []any
		if lineaID != nil {
			dataQuery = `
SELECT id, linea_id, dia_semana, nombre,
       hora_inicio::TEXT, hora_fin::TEXT, color, activo,
       renovacion_semanal, vigente_desde::TEXT, creado_en
FROM public.turno_dia
WHERE linea_id = $1 AND vigente_desde = $2::DATE
ORDER BY dia_semana, hora_inicio`
			dataArgs = []any{*lineaID, vigenteDesde}
		} else {
			dataQuery = `
SELECT id, linea_id, dia_semana, nombre,
       hora_inicio::TEXT, hora_fin::TEXT, color, activo,
       renovacion_semanal, vigente_desde::TEXT, creado_en
FROM public.turno_dia
WHERE linea_id IS NULL AND vigente_desde = $1::DATE
ORDER BY dia_semana, hora_inicio`
			dataArgs = []any{vigenteDesde}
		}

		rows, err := pool.Query(ctx, dataQuery, dataArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var result []TurnoDia
		renovacionSemanal := true
		for rows.Next() {
			var t TurnoDia
			t.PlantaID = plantaID
			if err := rows.Scan(&t.ID, &t.LineaID, &t.DiaSemana,
				&t.Nombre, &t.HoraInicio, &t.HoraFin, &t.Color, &t.Activo,
				&t.RenovacionSemanal, &t.VigenteDesdé, &t.CreadoEn); err != nil {
				continue
			}
			renovacionSemanal = t.RenovacionSemanal
			result = append(result, t)
		}

		if result == nil {
			result = []TurnoDia{}
		}
		c.JSON(http.StatusOK, gin.H{
			"data":               result,
			"vigente_desde":      vigenteDesde,
			"renovacion_semanal": renovacionSemanal,
			"versiones":          versiones,
		})
	})

	// PUT /turno-dias
	g.PUT("", func(c *gin.Context) {
		var body struct {
			PlantaID          int        `json:"planta_id"`
			LineaID           *int       `json:"linea_id"`
			VigenteDesdé      string     `json:"vigente_desde"`
			RenovacionSemanal bool       `json:"renovacion_semanal"`
			Turnos            []TurnoDia `json:"turnos"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.PlantaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "planta_id y turnos requeridos"})
			return
		}
		if body.VigenteDesdé == "" {
			body.VigenteDesdé = time.Now().Format("2006-01-02")
		}

		vigente, err := time.Parse("2006-01-02", body.VigenteDesdé)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "vigente_desde con formato invalido (YYYY-MM-DD)"})
			return
		}
		hoy := time.Now().Truncate(24 * time.Hour)
		if vigente.Before(hoy) {
			c.JSON(http.StatusConflict, gin.H{
				"error":         "cambio_retroactivo",
				"mensaje":       "No se puede modificar configuraciones con fecha anterior a hoy.",
				"vigente_desde": body.VigenteDesdé,
				"hoy":           hoy.Format("2006-01-02"),
			})
			return
		}

		pool, err := mgr.Get(c.Request.Context(), body.PlantaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(ctx)

		if body.LineaID != nil {
			_, err = tx.Exec(ctx,
				`DELETE FROM public.turno_dia WHERE linea_id = $1 AND vigente_desde >= $2::DATE`,
				*body.LineaID, body.VigenteDesdé)
		} else {
			_, err = tx.Exec(ctx,
				`DELETE FROM public.turno_dia WHERE linea_id IS NULL AND vigente_desde >= $1::DATE`,
				body.VigenteDesdé)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, t := range body.Turnos {
			color := t.Color
			if color == "" {
				color = "#6366f1"
			}
			_, err := tx.Exec(ctx, `
INSERT INTO public.turno_dia
  (linea_id, dia_semana, nombre, hora_inicio, hora_fin, color, activo, renovacion_semanal, vigente_desde)
VALUES ($1, $2, $3, $4::TIME, $5::TIME, $6, TRUE, $7, $8::DATE)
`, body.LineaID, t.DiaSemana, t.Nombre, t.HoraInicio, t.HoraFin, color, body.RenovacionSemanal, body.VigenteDesdé)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if notifier != nil {
			savedPlantaID := body.PlantaID
			savedVigente := body.VigenteDesdé
			go func() {
				ctx2 := context.Background()
				var empresaID int64
				if err := masterDB.QueryRow(ctx2,
					"SELECT empresa_id FROM config.plantas WHERE id=$1", savedPlantaID,
				).Scan(&empresaID); err != nil || empresaID == 0 {
					return
				}
				pPool, err := mgr.Get(ctx2, savedPlantaID)
				if err != nil {
					return
				}
				rows, err := pPool.Query(ctx2, `
					SELECT DISTINCT ON (nombre, hora_inicio, hora_fin)
						id, nombre, hora_inicio::TEXT, hora_fin::TEXT, activo
					FROM public.turno_dia
					WHERE vigente_desde = $1::DATE AND activo = true
					ORDER BY nombre, hora_inicio, hora_fin, id
				`, savedVigente)
				if err != nil {
					return
				}
				defer rows.Close()
				type syncRecord struct {
					ID         int    `json:"id"`
					Nombre     string `json:"nombre"`
					HoraInicio string `json:"hora_inicio"`
					HoraFin    string `json:"hora_fin"`
					PlantaID   int    `json:"planta_id"`
					Activo     bool   `json:"activo"`
				}
				var records []syncRecord
				for rows.Next() {
					var sr syncRecord
					sr.PlantaID = savedPlantaID
					if scanErr := rows.Scan(&sr.ID, &sr.Nombre, &sr.HoraInicio, &sr.HoraFin, &sr.Activo); scanErr == nil {
						records = append(records, sr)
					}
				}
				if records == nil {
					records = []syncRecord{}
				}
				notifier.Broadcast(ctx2, "turnos", empresaID, records)
			}()
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "vigente_desde": body.VigenteDesdé})
	})
}

// RegisterTurnosSimpleRoutes expone GET /turnos como lista plana
// de registros turno_dia (reemplaza el legacy config.turnos).
// Acepta ?linea_id=X | ?planta_id=X | empresa del JWT como fallback.
func RegisterTurnosSimpleRoutes(
	r *gin.RouterGroup,
	masterDB *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	mw *ports.JWTMiddleware,
) {
	type turnoSimple struct {
		ID         int    `json:"id"`
		Nombre     string `json:"nombre"`
		HoraInicio string `json:"hora_inicio"`
		HoraFin    string `json:"hora_fin"`
		DiaSemana  int    `json:"dia_semana"`
		Activo     bool   `json:"activo"`
	}

	g := r.Group("/turnos", mw.Auth())

	g.GET("", func(c *gin.Context) {
		ctx := c.Request.Context()
		var result []turnoSimple

		queryPool := func(pool *pgxpool.Pool, scope string) error {
			var q string
			var args []any
			if scope != "" {
				// scope = linea_id filter
				q = `SELECT id, nombre, hora_inicio::TEXT, hora_fin::TEXT, dia_semana, activo
					  FROM public.turno_dia WHERE linea_id = $1 AND activo=true ORDER BY dia_semana, nombre`
				k, _ := strconv.Atoi(scope)
				args = []any{k}
			} else {
				q = `SELECT id, nombre, hora_inicio::TEXT, hora_fin::TEXT, dia_semana, activo
					  FROM public.turno_dia WHERE activo=true ORDER BY dia_semana, nombre`
			}
			rows, err := pool.Query(ctx, q, args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var t turnoSimple
				if err := rows.Scan(&t.ID, &t.Nombre, &t.HoraInicio, &t.HoraFin, &t.DiaSemana, &t.Activo); err == nil {
					result = append(result, t)
				}
			}
			return nil
		}

		// Priority: linea_id > planta_id > empresa from JWT
		if lidStr := c.Query("linea_id"); lidStr != "" {
			lid, _ := strconv.ParseInt(lidStr, 10, 64)
			pool, _, err := resolvePlantaPool(ctx, masterDB, mgr, lid)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			_ = queryPool(pool, lidStr)
		} else if pidStr := c.Query("planta_id"); pidStr != "" {
			pid, _ := strconv.Atoi(pidStr)
			pool, err := mgr.Get(ctx, pid)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			_ = queryPool(pool, "")
		} else {
			// Obtener empresa_id del JWT y recorrer todas sus plantas
			var empresaID int64
			if eid, exists := c.Get("empresa_id"); exists {
				if e, ok := eid.(int64); ok {
					empresaID = e
				}
			}
			rows2, err := masterDB.Query(ctx,
				`SELECT id FROM config.plantas WHERE empresa_id = $1 AND activo=true`, empresaID)
			if err == nil {
				defer rows2.Close()
				for rows2.Next() {
					var pid int
					if err2 := rows2.Scan(&pid); err2 != nil {
						continue
					}
					pool, poolErr := mgr.Get(ctx, pid)
					if poolErr == nil {
						_ = queryPool(pool, "")
					}
				}
			}
		}

		if result == nil {
			result = []turnoSimple{}
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})
}
