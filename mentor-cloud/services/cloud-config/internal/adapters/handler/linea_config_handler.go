package handler

import (
	"log"
	"net/http"
	"strconv"

	"cloud-config/internal/adapters"
	"cloud-config/internal/ports"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// validModes son los modos de operación válidos para una línea.
var validModes = map[string]bool{"textil": true}

// modeVelUnit mapea mode → vel_unit string para responder en el API.
var modeVelUnit = map[string]string{
	"textil": "uh",
}

// RegisterLineaConfigRoutes registra los endpoints de configuración de modo de línea.
//
// GET /linea-config?linea_id=X → {mode, vel_unit}
// PUT /linea-config             → body {linea_id, mode}
func RegisterLineaConfigRoutes(r *gin.RouterGroup, db *pgxpool.Pool, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	g := r.Group("/linea-config", mw.Auth())

	// GET /linea-config?linea_id=X
	g.GET("", func(c *gin.Context) {
		lineaID, err := strconv.ParseInt(c.Query("linea_id"), 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
			return
		}

		var mode string
		err = db.QueryRow(c.Request.Context(),
			`SELECT COALESCE(mode, 'textil') FROM config.lineas WHERE id = $1`, lineaID,
		).Scan(&mode)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "línea no encontrada"})
			return
		}

		velUnit := modeVelUnit[mode]
		if velUnit == "" {
			velUnit = "uh"
		}
		c.JSON(http.StatusOK, gin.H{
			"linea_id": lineaID,
			"mode":     mode,
			"vel_unit": velUnit,
		})
	})

	// PUT /linea-config — body: {linea_id, mode}
	g.PUT("", func(c *gin.Context) {
		var body struct {
			LineaID int64  `json:"linea_id"`
			Mode    string `json:"mode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.LineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id y mode requeridos"})
			return
		}
		if !validModes[body.Mode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode debe ser 'textil'"})
			return
		}

		tag, err := db.Exec(c.Request.Context(),
			`UPDATE config.lineas SET mode = $1 WHERE id = $2`, body.Mode, body.LineaID)
		if err != nil {
			log.Printf("[linea-config] update mode error linea=%d: %v", body.LineaID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error actualizando modo"})
			return
		}
		if tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "línea no encontrada"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":       true,
			"linea_id": body.LineaID,
			"mode":     body.Mode,
			"vel_unit": modeVelUnit[body.Mode],
		})
	})
}

// RegisterLineaConfigEdgeRoutes expone PUT /edge/linea-config (path post-strip del gateway con strip="/api/v1")
// para que el enviador del edge pueda sincronizar el modo de la línea al cloud.
// No requiere JWT: la autenticación la hace el cloud-gateway (APIKey).
// Usa X-Linea-ID (cloud linea_id) seteado por el gateway, ignorando el body linea_id que es local al edge.
func RegisterLineaConfigEdgeRoutes(r *gin.Engine, db *pgxpool.Pool, notifier *adapters.SyncNotifier) {
	r.PUT("/edge/linea-config", func(c *gin.Context) {
		var body struct {
			Mode string `json:"mode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "body inválido"})
			return
		}
		if !validModes[body.Mode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode inválido"})
			return
		}

		// X-Linea-ID es el cloud linea_id seteado por el gateway desde el registro del dispositivo.
		lineaIDStr := c.GetHeader("X-Linea-ID")
		lineaID, err := strconv.ParseInt(lineaIDStr, 10, 64)
		if err != nil || lineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Linea-ID header requerido"})
			return
		}

		tag, err := db.Exec(c.Request.Context(),
			`UPDATE config.lineas SET mode = $1 WHERE id = $2`, body.Mode, lineaID)
		if err != nil {
			log.Printf("[edge/linea-config] update mode error linea=%d: %v", lineaID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error actualizando modo"})
			return
		}
		if tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "línea no encontrada"})
			return
		}
		log.Printf("[edge/linea-config] mode synced linea=%d mode=%s", lineaID, body.Mode)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}
