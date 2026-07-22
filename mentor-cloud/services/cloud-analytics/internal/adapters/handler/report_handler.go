package handler

import (
"net/http"

"github.com/gin-gonic/gin"
)

func RegisterReportRoutes(rg *gin.RouterGroup, mw gin.HandlerFunc) {
g := rg.Group("/reportes", mw)
g.GET("", listReportes)

// Rutas directas para que coincidan con el contrato del frontend
excel := rg.Group("/excel", mw)
excel.GET("/export", exportExcel)
excel.POST("/import", importExcel)
}

func listReportes(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{
"data": []gin.H{},
})
}

func exportExcel(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{
"message": "Funcionalidad de exportacion en desarrollo",
})
}

func importExcel(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{
"message": "Funcionalidad de importacion en desarrollo",
})
}
