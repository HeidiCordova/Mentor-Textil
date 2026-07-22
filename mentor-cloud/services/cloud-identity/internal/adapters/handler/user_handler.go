package handler

import (
	"cloud-identity/internal/adapters"
	"cloud-identity/internal/application"
	"cloud-identity/internal/domain"
	"cloud-identity/internal/ports"
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserHandler struct {
	svc      *application.UserService
	notifier *adapters.SyncNotifier
}

func RegisterUserRoutes(r *gin.RouterGroup, svc *application.UserService, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	h := &UserHandler{svc: svc, notifier: notifier}
	g := r.Group("/usuarios", mw.Auth())
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)

	r.GET("/auth/me", mw.Auth(), h.Me)
}

// resolveEmpresaID returns empresa_id from JWT context (trusted).
// Falls back to query param only for global admins without empresa in token.
func resolveEmpresaID(c *gin.Context) *int {
	// ADMIN manages the whole system — no forced filter
	if role, _ := c.Get("user_role"); role == "ADMIN" {
		if v := c.Query("empresa_id"); v != "" {
			id, _ := strconv.Atoi(v)
			if id > 0 {
				return &id
			}
		}
		return nil
	}
	// All other roles are scoped to their JWT empresa_id
	if v, exists := c.Get("empresa_id"); exists {
		if eid, ok := v.(int64); ok && eid > 0 {
			id := int(eid)
			return &id
		}
	}
	return nil
}

func (h *UserHandler) List(c *gin.Context) {
	empresaID := resolveEmpresaID(c)
	users, err := h.svc.List(c.Request.Context(), empresaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":    users,
		"total":   len(users),
		"page":    1,
		"perPage": len(users),
	})
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	user, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	c.JSON(http.StatusOK, user)
}

type createUserRequest struct {
	Username  string `json:"username" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Nombre    string `json:"nombre" binding:"required"`
	Apellido  string `json:"apellido"`
	Password  string `json:"password" binding:"required"`
	RolID     *int   `json:"rol_id"`
	EmpresaID *int   `json:"empresa_id"`
	PlantaID  *int   `json:"planta_id"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Enforce empresa_id from JWT — users cannot be created in another company
	jwtEmpresa := resolveEmpresaID(c)
	if jwtEmpresa != nil {
		req.EmpresaID = jwtEmpresa
	}

	u := &domain.Usuario{
		Username:  req.Username,
		Email:     req.Email,
		Nombre:    req.Nombre,
		Apellido:  req.Apellido,
		RolID:     req.RolID,
		EmpresaID: req.EmpresaID,
		PlantaID:  req.PlantaID,
	}

	if err := h.svc.Create(c.Request.Context(), u, req.Password); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "el nombre de usuario o correo ya est\u00e1 registrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.syncUsuariosForEmpresa(c, u.EmpresaID)
	c.JSON(http.StatusCreated, u)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var u domain.Usuario
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u.ID = id

	if err := h.svc.Update(c.Request.Context(), &u); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.syncUsuariosForEmpresa(c, u.EmpresaID)
	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.syncUsuarios(c)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *UserHandler) syncUsuarios(c *gin.Context) {
	if h.notifier == nil {
		return
	}
	var empresaInt int64
	if eid := resolveEmpresaID(c); eid != nil {
		empresaInt = int64(*eid)
	}
	if empresaInt <= 0 {
		if v, ok := c.Get("empresa_id"); ok {
			if eid, ok := v.(int64); ok {
				empresaInt = eid
			}
		}
	}
	if empresaInt <= 0 {
		return
	}
	eidInt := int(empresaInt)
	all, _ := h.svc.List(c.Request.Context(), &eidInt)
	go h.notifier.Broadcast(context.Background(), "usuarios", empresaInt, all)
}

// syncUsuariosForEmpresa dispara el sync para la empresa indicada.
// Cuando el admin crea/edita un usuario de otra empresa, el JWT no lleva
// empresa_id, por lo que se usa directamente el del usuario afectado.
// Si el usuario es global (empresa_id=NULL), se sincroniza a todas las empresas.
func (h *UserHandler) syncUsuariosForEmpresa(c *gin.Context, empresaID *int) {
	if h.notifier == nil {
		return
	}
	// Si el JWT ya lleva empresa, delega al método estándar
	if eid := resolveEmpresaID(c); eid != nil {
		h.syncUsuarios(c)
		return
	}
	// Usuario con empresa asignada: sync solo a esa empresa
	if empresaID != nil && *empresaID > 0 {
		eidInt := *empresaID
		all, _ := h.svc.List(c.Request.Context(), &eidInt)
		go h.notifier.Broadcast(context.Background(), "usuarios", int64(eidInt), all)
		return
	}
	// Admin global (empresa_id=NULL): sincronizar a todas las empresas que tengan usuarios
	allUsers, err := h.svc.List(c.Request.Context(), nil)
	if err != nil || len(allUsers) == 0 {
		return
	}
	seen := make(map[int]bool)
	for _, u := range allUsers {
		if u.EmpresaID == nil || *u.EmpresaID <= 0 {
			continue
		}
		if seen[*u.EmpresaID] {
			continue
		}
		seen[*u.EmpresaID] = true
		eid := *u.EmpresaID
		usersForEmpresa, _ := h.svc.List(c.Request.Context(), &eid)
		go h.notifier.Broadcast(context.Background(), "usuarios", int64(eid), usersForEmpresa)
	}
}

func (h *UserHandler) Me(c *gin.Context) {
	userID := c.GetInt("user_id")
	profile, err := h.svc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "usuario no encontrado"})
		return
	}
	c.JSON(http.StatusOK, profile)
}
