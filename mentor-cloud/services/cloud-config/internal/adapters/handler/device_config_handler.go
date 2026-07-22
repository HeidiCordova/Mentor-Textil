package handler

import (
	"cloud-config/internal/adapters/repository"
	"cloud-config/internal/domain"
	"cloud-config/internal/ports"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterDeviceConfigRoutes(r *gin.RouterGroup, dcRepo *repository.DeviceConfigRepo, dispRepo *repository.DispositivoRepo) {
	edge := r.Group("/api/v1/config", ports.APIKeyAuth())

	// Jetson enviador pulls pending config
	edge.GET("/:device_id/pending", func(c *gin.Context) {
		deviceID := c.Param("device_id")
		disp, err := dispRepo.FindByDeviceID(c.Request.Context(), deviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}

		_ = dispRepo.UpdatePing(c.Request.Context(), deviceID)

		dc, err := dcRepo.GetPending(c.Request.Context(), disp.ID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"pending": false})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pending": true,
			"version": dc.Version,
			"payload": json.RawMessage(dc.Payload),
		})
	})

	// Jetson enviador acknowledges config applied
	edge.POST("/:device_id/ack", func(c *gin.Context) {
		deviceID := c.Param("device_id")
		var req domain.ConfigAck
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		disp, err := dispRepo.FindByDeviceID(c.Request.Context(), deviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}

		if err := dcRepo.MarkApplied(c.Request.Context(), disp.ID, req.Version); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Jetson enviador pushes local config changes
	edge.POST("/:device_id/sync", func(c *gin.Context) {
		deviceID := c.Param("device_id")
		var req domain.ConfigSync
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		disp, err := dispRepo.FindByDeviceID(c.Request.Context(), deviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}

		_ = dispRepo.UpdatePing(c.Request.Context(), deviceID)

		currentVersion, _ := dcRepo.GetLatestVersion(c.Request.Context(), disp.ID)
		newVersion := currentVersion + 1

		dc := &domain.DeviceConfig{
			DispositivoID: disp.ID,
			Version:       newVersion,
			Payload:       req.Payload,
			Aplicado:      true,
		}

		if err := dcRepo.Create(c.Request.Context(), dc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"version": newVersion})
	})
}
