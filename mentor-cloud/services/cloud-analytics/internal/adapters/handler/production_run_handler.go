package handler

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"mentor.local/shared/multitenancy"
)

type ProductionRunHandler struct {
	repo ports.ProductionRunRepository
	pr   *multitenancy.PoolResolver
}

func RegisterProductionRunRoutes(rg *gin.RouterGroup, repo ports.ProductionRunRepository, pr *multitenancy.PoolResolver, mw gin.HandlerFunc) {
	h := &ProductionRunHandler{repo: repo, pr: pr}
	g := rg.Group("/production-runs", mw)
	g.GET("", h.List)
	g.POST("", h.Upsert)
	g.DELETE("/:runId", h.Delete)
}

func (h *ProductionRunHandler) List(c *gin.Context) {
	f := parseProductionRunFilter(c)
	data, err := h.repo.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *ProductionRunHandler) Upsert(c *gin.Context) {
	var req domain.UpsertProductionRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	scope, hasScope := multitenancy.ScopeFrom(c.Request.Context())
	if !hasScope || scope.LineaID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id and planta_id scope are required"})
		return
	}
	if req.LineaID != nil && *req.LineaID != scope.LineaID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body linea_id does not match request scope"})
		return
	}
	lineaID := scope.LineaID
	req.LineaID = &lineaID

	// Generar run_id si no se proporcionó (ej: tablet cloud crea nuevo run)
	if req.RunID == "" {
		req.RunID = uuid.New().String()
	}

	// Asignar device_id si está vacío
	edgeDeviceID, err := h.findEdgeDeviceID(c.Request.Context(), lineaID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "the selected line has no registered edge device",
		})
		return
	}
	req.DeviceID = edgeDeviceID

	// Llenar empresa_id desde JWT si no viene en el body
	if req.EmpresaID == nil {
		if v, ok := c.Get("empresa_id"); ok {
			switch eid := v.(type) {
			case float64:
				i := int(eid)
				req.EmpresaID = &i
			case string:
				if i, err := strconv.Atoi(eid); err == nil {
					req.EmpresaID = &i
				}
			}
		}
	}

	pr, err := h.repo.Upsert(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Encolar pending_command para propagar al edge si el upsert vino del cloud tablet
	h.enqueueEdgeSync(c.Request.Context(), pr, edgeDeviceID)

	c.JSON(http.StatusOK, []domain.ProductionRun{*pr})
}

func (h *ProductionRunHandler) Delete(c *gin.Context) {
	runID := c.Param("runId")
	if err := h.repo.Delete(c.Request.Context(), runID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func parseProductionRunFilter(c *gin.Context) ports.ProductionRunFilter {
	f := ports.ProductionRunFilter{}

	if v := c.Query("device_id"); v != "" {
		f.DeviceID = v
	}
	if v := c.Query("empresa_id"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.EmpresaID = &i
		}
	}
	if v := c.Query("planta_id"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.PlantaID = &i
		}
	}
	if v := c.Query("linea_id"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.LineaID = &i
		}
	}
	if v := c.Query("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Since = &t
		}
	}
	if v := c.Query("until"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Until = &t
		}
	}
	if v := c.Query("limit"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			f.Limit = i
		}
	}
	return f
}

// enqueueEdgeSync inserta un pending_command para que el enviador del edge
// aplique la asignación de producto realizada desde el cloud tablet.
func (h *ProductionRunHandler) findEdgeDeviceID(ctx context.Context, lineaID int) (string, error) {
	var deviceID string
	err := h.pr.Master().QueryRow(ctx,
		`SELECT device_id
		   FROM gateway.device_registry
		  WHERE linea_id=$1
		  ORDER BY active DESC, last_seen_at DESC NULLS LAST
		  LIMIT 1`,
		lineaID,
	).Scan(&deviceID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(deviceID) == "" {
		return "", fmt.Errorf("empty device_id")
	}
	return deviceID, nil
}

func (h *ProductionRunHandler) enqueueEdgeSync(
	ctx context.Context,
	pr *domain.ProductionRun,
	edgeDeviceID string,
) {
	pool, schema, err := h.pr.Resolve(ctx)
	if err != nil {
		log.Printf("[production_run_handler] warn: resolve for pending_command: %v", err)
		return
	}
	if schema == "" {
		return // sin tenant, no hay edge al que propagar
	}

	// Buscar el device_id del edge para esta línea (en la tabla master)
	scope, ok := multitenancy.ScopeFrom(ctx)
	if !ok {
		return
	}
	if edgeDeviceID == "" {
		edgeDeviceID, err = h.findEdgeDeviceID(ctx, scope.LineaID)
		if err != nil {
			log.Printf("[production_run_handler] warn: no edge device for linea %d: %v", scope.LineaID, err)
			return
		}
	}

	pcTbl := multitenancy.Tbl(schema, "analytics.pending_commands")
	payload, _ := json.Marshal(map[string]interface{}{
		"run_id":      pr.RunID,
		"producto_id": pr.ProductoID,
		"sku":         pr.SKU,
		"nombre":      pr.Nombre,
		"started_at":  pr.StartedAt,
		"ended_at":    pr.EndedAt,
	})
	if _, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (device_id, command, payload)
		VALUES ($1, 'upsert_production_run', $2::jsonb)
	`, pcTbl), edgeDeviceID, string(payload)); err != nil {
		log.Printf("[production_run_handler] warn: enqueue pending_command: %v", err)
	}
}
