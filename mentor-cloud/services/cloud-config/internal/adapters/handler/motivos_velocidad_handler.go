package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"cloud-config/internal/adapters"
	"cloud-config/internal/ports"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"mentor.local/shared/multitenancy"
)

// RegisterMotivosVelocidadRoutes registra los endpoints de motivos de velocidad nominal.
//
// GET    /motivos-velocidad?linea_id=X  → listar motivos activos
// POST   /motivos-velocidad             → crear motivo
// PUT    /motivos-velocidad/:id         → editar motivo
// DELETE /motivos-velocidad/:id         → eliminar (soft-delete)
func RegisterMotivosVelocidadRoutes(
	r *gin.RouterGroup,
	db *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	mw *ports.JWTMiddleware,
	notifier *adapters.SyncNotifier,
) {
	g := r.Group("/motivos-velocidad", mw.Auth())

	// ── GET ─────────────────────────────────────────────────────────────
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
		rows, err := pool.Query(c.Request.Context(), fmt.Sprintf(
			`SELECT id, texto, activo, orden FROM %s.motivos_velocidad ORDER BY orden, id`, schema))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		type Row struct {
			ID     int    `json:"id"`
			Texto  string `json:"texto"`
			Activo bool   `json:"activo"`
			Orden  int    `json:"orden"`
		}
		var result []Row
		for rows.Next() {
			var rv Row
			if err := rows.Scan(&rv.ID, &rv.Texto, &rv.Activo, &rv.Orden); err == nil {
				result = append(result, rv)
			}
		}
		if result == nil {
			result = []Row{}
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	// ── POST ─────────────────────────────────────────────────────────────
	g.POST("", func(c *gin.Context) {
		lineaID, err := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		body, _ := io.ReadAll(c.Request.Body)
		var req struct {
			Texto string `json:"texto"`
			Orden int    `json:"orden"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.Texto == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "texto requerido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), db, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		var id int
		err = pool.QueryRow(c.Request.Context(), fmt.Sprintf(
			`INSERT INTO %s.motivos_velocidad (texto, orden) VALUES ($1, $2) RETURNING id`, schema),
			req.Texto, req.Orden,
		).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		go broadcastMotivos(db, mgr, notifier, lineaID)
		c.JSON(http.StatusCreated, gin.H{"id": id, "ok": true})
	})

	// ── PUT /:id ──────────────────────────────────────────────────────────
	g.PUT("/:id", func(c *gin.Context) {
		lineaID, err := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}
		body, _ := io.ReadAll(c.Request.Body)
		var req struct {
			Texto  string `json:"texto"`
			Activo *bool  `json:"activo"`
			Orden  *int   `json:"orden"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), db, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		// Construir UPDATE dinámico
		sets := []string{}
		args := []interface{}{}
		idx := 1
		if req.Texto != "" {
			sets = append(sets, fmt.Sprintf("texto = $%d", idx))
			args = append(args, req.Texto)
			idx++
		}
		if req.Activo != nil {
			sets = append(sets, fmt.Sprintf("activo = $%d", idx))
			args = append(args, *req.Activo)
			idx++
		}
		if req.Orden != nil {
			sets = append(sets, fmt.Sprintf("orden = $%d", idx))
			args = append(args, *req.Orden)
			idx++
		}
		if len(sets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nada que actualizar"})
			return
		}
		args = append(args, itemID)
		q := fmt.Sprintf("UPDATE %s.motivos_velocidad SET %s WHERE id = $%d",
			schema, join(sets, ", "), idx)
		cmd, err := pool.Exec(c.Request.Context(), q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cmd.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "motivo no encontrado"})
			return
		}
		go broadcastMotivos(db, mgr, notifier, lineaID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// ── DELETE /:id ───────────────────────────────────────────────────────
	g.DELETE("/:id", func(c *gin.Context) {
		lineaID, err := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}
		pool, schema, err := resolvePlantaPool(c.Request.Context(), db, mgr, lineaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		cmd, err := pool.Exec(c.Request.Context(), fmt.Sprintf(
			`UPDATE %s.motivos_velocidad SET activo = false WHERE id = $1`, schema), itemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if cmd.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "motivo no encontrado"})
			return
		}
		go broadcastMotivos(db, mgr, notifier, lineaID)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

// join concatena strings con separador (evita importar strings solo para esto).
func join(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	r := ss[0]
	for _, s := range ss[1:] {
		r += sep + s
	}
	return r
}

// broadcastMotivos envía la lista completa de motivos activos al edge como comando de sync.
func broadcastMotivos(
	db *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	notifier *adapters.SyncNotifier,
	lineaID int64,
) {
	if notifier == nil {
		return
	}
	ctx := context.Background()
	var empresaID int64
	if err := db.QueryRow(ctx,
		`SELECT p.empresa_id FROM config.lineas l
		 JOIN config.plantas p ON p.id = l.planta_id
		 WHERE l.id = $1`, lineaID,
	).Scan(&empresaID); err != nil || empresaID == 0 {
		return
	}
	pool, schema, err := resolvePlantaPool(ctx, db, mgr, lineaID)
	if err != nil {
		return
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(
		`SELECT id, texto, activo, orden FROM %s.motivos_velocidad ORDER BY orden, id`, schema))
	if err != nil {
		return
	}
	defer rows.Close()
	type item struct {
		ID     int    `json:"id"`
		Texto  string `json:"texto"`
		Activo bool   `json:"activo"`
		Orden  int    `json:"orden"`
	}
	var items []item
	for rows.Next() {
		var it item
		if rows.Scan(&it.ID, &it.Texto, &it.Activo, &it.Orden) == nil {
			items = append(items, it)
		}
	}
	if len(items) > 0 {
		notifier.Broadcast(ctx, "motivos_velocidad", empresaID, items)
	}
}
