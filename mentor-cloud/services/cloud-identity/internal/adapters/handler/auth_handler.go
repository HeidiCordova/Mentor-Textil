package handler

import (
	"cloud-identity/internal/application"
	"cloud-identity/internal/domain"
	"cloud-identity/internal/ports"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *application.AuthService
}

func RegisterAuthRoutes(r *gin.RouterGroup, auth *application.AuthService, mw *ports.JWTMiddleware) {
	h := &AuthHandler{auth: auth}
	g := r.Group("/auth")
	g.POST("/login", h.Login)
	g.POST("/logout", mw.Auth(), h.Logout)
	g.POST("/refresh", h.Refresh)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos"})
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}
	_ = h.auth.Logout(c.Request.Context(), userID.(int))
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req domain.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refreshToken requerido"})
		return
	}

	resp, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
