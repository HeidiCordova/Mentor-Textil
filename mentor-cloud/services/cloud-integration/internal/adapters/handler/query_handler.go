package handler

import (
	"net/http"
	"strconv"

	"cloud-integration/internal/domain"
	"cloud-integration/internal/ports"

	"github.com/gin-gonic/gin"
)

func RegisterQueryRoutes(v1 *gin.RouterGroup, repo ports.QueryRepository, apiKeyMw *ports.APIKeyMiddleware) {
	g := v1.Group("/integration", apiKeyMw.Auth())
	g.GET("/oee/latest", latestOEE(repo))
	g.GET("/oee/snapshots", listSnapshots(repo))
	g.GET("/paradas", listParadas(repo))
	g.GET("/energy/snapshots", listEnergy(repo))
}

func latestOEE(repo ports.QueryRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		if !ports.HasScope(c, "oee:read") {
			c.JSON(http.StatusForbidden, gin.H{"error": "scope oee:read requerido"})
			return
		}
		empresaID, _ := ports.EmpresaIDFromCtx(c)
		snap, err := repo.LatestSnapshot(c.Request.Context(), lineaID, empresaID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "sin datos"})
			return
		}
		c.JSON(http.StatusOK, snap)
	}
}

func listSnapshots(repo ports.QueryRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		if !ports.HasScope(c, "snapshots:read") {
			c.JSON(http.StatusForbidden, gin.H{"error": "scope snapshots:read requerido"})
			return
		}
		limit := 100
		if v := c.Query("limite"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 {
				limit = n
			}
		}
		empresaID, _ := ports.EmpresaIDFromCtx(c)
		snaps, err := repo.ListSnapshots(c.Request.Context(), lineaID, empresaID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if snaps == nil {
			snaps = []domain.OEESnapshot{}
		}
		c.JSON(http.StatusOK, snaps)
	}
}

func listParadas(repo ports.QueryRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		lineaID, err := strconv.Atoi(c.Query("linea_id"))
		if err != nil || lineaID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}
		if !ports.HasScope(c, "paradas:read") {
			c.JSON(http.StatusForbidden, gin.H{"error": "scope paradas:read requerido"})
			return
		}
		limit := 200
		if v := c.Query("limite"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 {
				limit = n
			}
		}
		empresaID, _ := ports.EmpresaIDFromCtx(c)
		paradas, err := repo.ListParadas(c.Request.Context(), lineaID, empresaID, c.Query("desde"), c.Query("hasta"), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if paradas == nil {
			paradas = []domain.Parada{}
		}
		c.JSON(http.StatusOK, paradas)
	}
}

// listEnergy sirve snapshots de energía eléctrica del medidor MC60.
// Soporta paginación (limite + offset) para cargas masivas desde Power BI.
//
// GET /api/v1/integration/energy/snapshots
//
// Parámetros opcionales:
//
//	device_id  string   Filtrar por dispositivo Edge
//	meter_id   string   Filtrar por ID de medidor
//	desde      ISO8601  Inicio del rango temporal
//	hasta      ISO8601  Fin del rango temporal
//	limite     int      Registros por página (default 200, max 2000)
//	offset     int      Saltar N registros (para paginación)
func listEnergy(repo ports.QueryRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ports.HasScope(c, "energy:read") {
			c.JSON(http.StatusForbidden, gin.H{"error": "scope energy:read requerido"})
			return
		}

		limit := 200
		if v := c.Query("limite"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n > 0 {
				limit = n
			}
		}
		offset := 0
		if v := c.Query("offset"); v != "" {
			if n, e := strconv.Atoi(v); e == nil && n >= 0 {
				offset = n
			}
		}

		empresaID, _ := ports.EmpresaIDFromCtx(c)
		data, total, err := repo.ListEnergy(
			c.Request.Context(),
			empresaID,
			c.Query("device_id"),
			c.Query("meter_id"),
			c.Query("desde"),
			c.Query("hasta"),
			limit,
			offset,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if data == nil {
			data = []domain.EnergySnapshot{}
		}
		c.JSON(http.StatusOK, domain.EnergyPage{
			Data:   data,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}
