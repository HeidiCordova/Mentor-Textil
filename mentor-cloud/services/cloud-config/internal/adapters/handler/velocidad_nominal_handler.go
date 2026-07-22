package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud-config/internal/adapters"
	"cloud-config/internal/ports"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"mentor.local/shared/multitenancy"
)

// RegisterVelocidadNominalRoutes registra los endpoints de velocidad nominal.
//
// GET  /velocidad-nominal?linea_id=X         → productos con velocidad de la línea
// PUT  /velocidad-nominal                    → upsert + log del cambio
// GET  /velocidad-nominal/log?linea_id=X     → historial de cambios
func RegisterVelocidadNominalRoutes(r *gin.RouterGroup, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	g := r.Group("/velocidad-nominal", mw.Auth())

	g.GET("", func(c *gin.Context) {
		lineaID, err := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}

		pool, schema, err := resolvePlantaPool(c.Request.Context(), db, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		rows, err := pool.Query(c.Request.Context(), fmt.Sprintf(`
			SELECT vn.id, vn.producto_id, p.codigo, p.nombre, vn.velocidad_us, vn.factor_conv
			FROM %s.velocidad_nominal vn
			JOIN %s.productos p ON p.id = vn.producto_id
			WHERE p.activo = true
			ORDER BY p.codigo`, schema, schema))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type Row struct {
			ID          int     `json:"id"`
			LineaID     int64   `json:"linea_id"`
			ProductoID  int     `json:"producto_id"`
			SKU         string  `json:"sku"`
			Descripcion string  `json:"descripcion"`
			VelocidadUS float64 `json:"velocidad_us"`
			FactorConv  int     `json:"factor_conv"`
		}
		var result []Row
		for rows.Next() {
			var row Row
			row.LineaID = lineaID
			if err := rows.Scan(&row.ID, &row.ProductoID, &row.SKU, &row.Descripcion, &row.VelocidadUS, &row.FactorConv); err != nil {
				continue
			}
			result = append(result, row)
		}
		if result == nil {
			result = []Row{}
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	g.PUT("", func(c *gin.Context) {
		type Item struct {
			LineaID     int64   `json:"linea_id"`
			ProductoID  int     `json:"producto_id"`
			VelocidadUS float64 `json:"velocidad_us"`
			FactorConv  int     `json:"factor_conv"`
			Motivo      string  `json:"motivo"`
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error leyendo body"})
			return
		}

		var items []Item
		if json.Unmarshal(body, &items) != nil || len(items) == 0 {
			var single Item
			if json.Unmarshal(body, &single) != nil || single.LineaID == 0 || single.ProductoID == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
				return
			}
			items = []Item{single}
		}

		lineaID := items[0].LineaID
		pool, schema, err := resolvePlantaPool(c.Request.Context(), db, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		usuario := ""
		if claims, ok := c.Get("claims"); ok {
			if m, ok := claims.(map[string]interface{}); ok {
				if u, _ := m["username"].(string); u != "" {
					usuario = u
				}
			}
		}

		// Leer valores previos para el log
		type prevRow struct {
			VelocidadUS float64
			FactorConv  int
			SKU         string
		}
		prevMap := make(map[int]prevRow)
		prevRows, _ := pool.Query(c.Request.Context(), fmt.Sprintf(`
			SELECT vn.producto_id, vn.velocidad_us, vn.factor_conv, p.codigo
			FROM %s.velocidad_nominal vn
			JOIN %s.productos p ON p.id = vn.producto_id`, schema, schema))
		if prevRows != nil {
			for prevRows.Next() {
				var pid int
				var pr prevRow
				if prevRows.Scan(&pid, &pr.VelocidadUS, &pr.FactorConv, &pr.SKU) == nil {
					prevMap[pid] = pr
				}
			}
			prevRows.Close()
		}

		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())

		upsertSQL := fmt.Sprintf(`
			INSERT INTO %s.velocidad_nominal (producto_id, velocidad_us, factor_conv)
			VALUES ($1, $2, $3)
			ON CONFLICT (producto_id)
			DO UPDATE SET velocidad_us = EXCLUDED.velocidad_us,
			              factor_conv  = EXCLUDED.factor_conv`, schema)

		logSQL := fmt.Sprintf(`
			INSERT INTO %s.velocidad_nominal_log
				(producto_id, sku, velocidad_us_anterior, velocidad_us_nueva,
				 factor_conv_anterior, factor_conv_nueva, motivo, usuario, origen)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'cloud')`, schema)

		for _, item := range items {
			if item.FactorConv <= 0 {
				item.FactorConv = 1
			}
			if _, err := tx.Exec(c.Request.Context(), upsertSQL,
				item.ProductoID, item.VelocidadUS, item.FactorConv); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			prev := prevMap[item.ProductoID]
			if prev.VelocidadUS == item.VelocidadUS && prev.FactorConv == item.FactorConv {
				continue
			}
			var vAnterior *float64
			var fcAnterior *int
			if prev.SKU != "" {
				vAnterior = &prev.VelocidadUS
				fcAnterior = &prev.FactorConv
			}
			var motivoVal *string
			if item.Motivo != "" {
				motivoVal = &item.Motivo
			}
			var usuarioVal *string
			if usuario != "" {
				usuarioVal = &usuario
			}
			if _, err := tx.Exec(c.Request.Context(), logSQL,
				item.ProductoID, prev.SKU,
				vAnterior, item.VelocidadUS,
				fcAnterior, item.FactorConv,
				motivoVal, usuarioVal,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if notifier != nil {
			go func() {
				ctx := context.Background()
				var empresaID int64
				if err := db.QueryRow(ctx,
					`SELECT p.empresa_id FROM config.lineas l
					 JOIN config.plantas p ON p.id = l.planta_id
					 WHERE l.id = $1`, lineaID,
				).Scan(&empresaID); err != nil || empresaID == 0 {
					return
				}
				vnPool, vnSchema, err := resolvePlantaPool(ctx, db, mgr, lineaID)
				if err != nil {
					return
				}
				rows, err := vnPool.Query(ctx, fmt.Sprintf(
					`SELECT producto_id, velocidad_us, factor_conv
					 FROM %s.velocidad_nominal ORDER BY producto_id`, vnSchema))
				if err != nil {
					return
				}
				defer rows.Close()
				type vnRecord struct {
					LineaID     int64   `json:"linea_id"`
					ProductoID  int     `json:"producto_id"`
					VelocidadUs float64 `json:"velocidad_us"`
					FactorConv  int     `json:"factor_conv"`
				}
				var records []vnRecord
				for rows.Next() {
					var v vnRecord
					v.LineaID = lineaID
					if rows.Scan(&v.ProductoID, &v.VelocidadUs, &v.FactorConv) == nil {
						records = append(records, v)
					}
				}
				if len(records) > 0 {
					notifier.Broadcast(ctx, "velocidad_nominal", empresaID, records)
				}
			}()
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "updated": len(items)})
	})

	// GET /velocidad-nominal/log?linea_id=X&limit=100
	g.GET("/log", func(c *gin.Context) {
		lineaID, err := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		if limit <= 0 || limit > 500 {
			limit = 100
		}

		pool, schema, err := resolvePlantaPool(c.Request.Context(), db, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		rows, err := pool.Query(c.Request.Context(), fmt.Sprintf(`
			SELECT id, producto_id, sku,
			       velocidad_us_anterior, velocidad_us_nueva,
			       factor_conv_anterior, factor_conv_nueva,
			       motivo, usuario, origen, cambiado_en
			FROM %s.velocidad_nominal_log
			ORDER BY cambiado_en DESC
			LIMIT $1`, schema), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type LogRow struct {
			ID                  int64    `json:"id"`
			ProductoID          int      `json:"producto_id"`
			SKU                 *string  `json:"sku"`
			VelocidadUSAnterior *float64 `json:"velocidad_us_anterior"`
			VelocidadUSNueva    float64  `json:"velocidad_us_nueva"`
			FactorConvAnterior  *int     `json:"factor_conv_anterior"`
			FactorConvNueva     int      `json:"factor_conv_nueva"`
			Motivo              *string  `json:"motivo"`
			Usuario             *string  `json:"usuario"`
			Origen              string   `json:"origen"`
			CambiadoEn          string   `json:"cambiado_en"`
		}
		var result []LogRow
		for rows.Next() {
			var row LogRow
			var ts time.Time
			if err := rows.Scan(
				&row.ID, &row.ProductoID, &row.SKU,
				&row.VelocidadUSAnterior, &row.VelocidadUSNueva,
				&row.FactorConvAnterior, &row.FactorConvNueva,
				&row.Motivo, &row.Usuario, &row.Origen, &ts,
			); err != nil {
				log.Printf("[velocidad-nominal/log] scan error linea=%d: %v", lineaID, err)
				continue
			}
			row.CambiadoEn = ts.Format(time.RFC3339)
			result = append(result, row)
		}
		if err := rows.Err(); err != nil {
			log.Printf("[velocidad-nominal/log] rows error linea=%d: %v", lineaID, err)
		}
		if result == nil {
			result = []LogRow{}
		}
		c.JSON(http.StatusOK, gin.H{"data": result, "linea_id": lineaID})
	})
}

// RegisterVelocidadNominalEdgeRoutes registra el endpoint PUT /edge/velocidad-nominal
// para dispositivos edge. No requiere JWT porque la autenticación es manejada por
// el cloud-gateway (API key del dispositivo). El gateway inyecta X-Linea-ID en el header.
func RegisterVelocidadNominalEdgeRoutes(r *gin.Engine, db *pgxpool.Pool, mgr *multitenancy.PlantaPoolManager, notifier *adapters.SyncNotifier) {
	r.PUT("/edge/velocidad-nominal", func(c *gin.Context) {
		lineaID, err := strconv.ParseInt(c.GetHeader("X-Linea-ID"), 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Linea-ID header requerido"})
			return
		}

		type Item struct {
			LineaID     int64   `json:"linea_id"`
			ProductoID  int     `json:"producto_id"`
			VelocidadUS float64 `json:"velocidad_us"`
			FactorConv  int     `json:"factor_conv"`
			Motivo      string  `json:"motivo,omitempty"`
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error leyendo body"})
			return
		}

		var items []Item
		if json.Unmarshal(body, &items) != nil || len(items) == 0 {
			var single Item
			if json.Unmarshal(body, &single) != nil || single.ProductoID == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
				return
			}
			items = []Item{single}
		}

		pool, schema, err := resolvePlantaPool(c.Request.Context(), db, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		type prevRow struct {
			VelocidadUS float64
			FactorConv  int
			SKU         string
		}
		prevMap := make(map[int]prevRow)
		prevRows, _ := pool.Query(c.Request.Context(), fmt.Sprintf(`
			SELECT vn.producto_id, vn.velocidad_us, vn.factor_conv, p.codigo
			FROM %s.velocidad_nominal vn
			JOIN %s.productos p ON p.id = vn.producto_id`, schema, schema))
		if prevRows != nil {
			for prevRows.Next() {
				var pid int
				var pr prevRow
				if prevRows.Scan(&pid, &pr.VelocidadUS, &pr.FactorConv, &pr.SKU) == nil {
					prevMap[pid] = pr
				}
			}
			prevRows.Close()
		}

		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback(c.Request.Context())

		upsertSQL := fmt.Sprintf(`
			INSERT INTO %s.velocidad_nominal (producto_id, velocidad_us, factor_conv)
			VALUES ($1, $2, $3)
			ON CONFLICT (producto_id)
			DO UPDATE SET velocidad_us = EXCLUDED.velocidad_us,
			              factor_conv  = EXCLUDED.factor_conv`, schema)

		logSQL := fmt.Sprintf(`
			INSERT INTO %s.velocidad_nominal_log
				(producto_id, sku, velocidad_us_anterior, velocidad_us_nueva,
				 factor_conv_anterior, factor_conv_nueva, motivo, usuario, origen)
			VALUES ($1,$2,$3,$4,$5,$6,$7,NULL,'edge')`, schema)

		for _, item := range items {
			if item.FactorConv <= 0 {
				item.FactorConv = 1
			}
			if _, err := tx.Exec(c.Request.Context(), upsertSQL,
				item.ProductoID, item.VelocidadUS, item.FactorConv); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			prev := prevMap[item.ProductoID]
			if prev.VelocidadUS == item.VelocidadUS && prev.FactorConv == item.FactorConv {
				continue
			}
			var vAnterior *float64
			var fcAnterior *int
			if prev.SKU != "" {
				vAnterior = &prev.VelocidadUS
				fcAnterior = &prev.FactorConv
			}
			var motivoVal *string
			if item.Motivo != "" {
				motivoVal = &item.Motivo
			}
			if _, err := tx.Exec(c.Request.Context(), logSQL,
				item.ProductoID, prev.SKU,
				vAnterior, item.VelocidadUS,
				fcAnterior, item.FactorConv,
				motivoVal,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if err := tx.Commit(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if notifier != nil {
			go func() {
				ctx := context.Background()
				var empresaID int64
				if err := db.QueryRow(ctx,
					`SELECT p.empresa_id FROM config.lineas l
					 JOIN config.plantas p ON p.id = l.planta_id
					 WHERE l.id = $1`, lineaID,
				).Scan(&empresaID); err != nil || empresaID == 0 {
					return
				}
				vnPool, vnSchema, err := resolvePlantaPool(ctx, db, mgr, lineaID)
				if err != nil {
					return
				}
				rows, err := vnPool.Query(ctx, fmt.Sprintf(
					`SELECT producto_id, velocidad_us, factor_conv
					 FROM %s.velocidad_nominal ORDER BY producto_id`, vnSchema))
				if err != nil {
					return
				}
				defer rows.Close()
				type vnRecord struct {
					LineaID     int64   `json:"linea_id"`
					ProductoID  int     `json:"producto_id"`
					VelocidadUs float64 `json:"velocidad_us"`
					FactorConv  int     `json:"factor_conv"`
				}
				var records []vnRecord
				for rows.Next() {
					var v vnRecord
					v.LineaID = lineaID
					if rows.Scan(&v.ProductoID, &v.VelocidadUs, &v.FactorConv) == nil {
						records = append(records, v)
					}
				}
				if len(records) > 0 {
					notifier.Broadcast(ctx, "velocidad_nominal", empresaID, records)
				}
			}()
		}

		c.JSON(http.StatusOK, gin.H{"ok": true, "updated": len(items)})
	})
}
