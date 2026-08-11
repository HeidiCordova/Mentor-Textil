package handler

import (
	"cloud-config/internal/adapters"
	"cloud-config/internal/adapters/repository"
	"cloud-config/internal/domain"
	"cloud-config/internal/ports"
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// resolveEmpresaID returns the empresa_id from the JWT context (injected by middleware).
// Falls back to query param only when the token has no empresa_id (global admin).
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

// ownsPlanta verifies the planta belongs to the request's empresa. Returns false if mismatch.
func ownsPlanta(c *gin.Context, repo *repository.PlantaRepo, plantaID int) bool {
	eid := resolveEmpresaID(c)
	if eid == nil {
		return true // global admin
	}
	p, err := repo.FindByID(c.Request.Context(), plantaID)
	if err != nil || p == nil {
		return false
	}
	return p.EmpresaID == *eid
}

func RegisterPlantaRoutes(r *gin.RouterGroup, repo *repository.PlantaRepo, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	g := r.Group("/plantas", mw.Auth())

	g.GET("", func(c *gin.Context) {
		empresaID := resolveEmpresaID(c)
		plantas, err := repo.FindAll(c.Request.Context(), empresaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data":    plantas,
			"total":   len(plantas),
			"page":    1,
			"perPage": len(plantas),
		})
	})

	g.GET("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		p, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "planta no encontrada"})
			return
		}
		c.JSON(http.StatusOK, p)
	})

	g.POST("", func(c *gin.Context) {
		var p domain.Planta
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Enforce empresa_id from JWT — cannot create resources in another company
		if eid := resolveEmpresaID(c); eid != nil {
			p.EmpresaID = *eid
		}
		p.Activo = true
		if err := repo.Create(c.Request.Context(), &p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if eid := empresaIDFromContext(c); eid > 0 && notifier != nil {
			go notifier.Broadcast(c.Request.Context(), "plantas_lineas", eid, nil)
		}
		c.JSON(http.StatusCreated, p)
	})

	g.PUT("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		if !ownsPlanta(c, repo, id) {
			c.JSON(http.StatusForbidden, gin.H{"error": "acceso denegado"})
			return
		}
		var p domain.Planta
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		p.ID = id
		if eid := resolveEmpresaID(c); eid != nil {
			p.EmpresaID = *eid
		}
		if err := repo.Update(c.Request.Context(), &p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if eid := empresaIDFromContext(c); eid > 0 && notifier != nil {
			go notifier.Broadcast(c.Request.Context(), "plantas_lineas", eid, nil)
		}
		c.JSON(http.StatusOK, p)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		if !ownsPlanta(c, repo, id) {
			c.JSON(http.StatusForbidden, gin.H{"error": "acceso denegado"})
			return
		}
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if eid := empresaIDFromContext(c); eid > 0 && notifier != nil {
			go notifier.Broadcast(c.Request.Context(), "plantas_lineas", eid, nil)
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}

func RegisterLineaRoutes(r *gin.RouterGroup, repo *repository.LineaRepo, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	g := r.Group("/lineas", mw.Auth())

	g.GET("", func(c *gin.Context) {
		var plantaID *int
		if v := c.Query("planta_id"); v != "" {
			id, _ := strconv.Atoi(v)
			plantaID = &id
		}
		lineas, err := repo.FindAll(c.Request.Context(), plantaID, resolveEmpresaID(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": lineas})
	})

	g.GET("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		l, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "linea no encontrada"})
			return
		}
		c.JSON(http.StatusOK, l)
	})

	g.POST("", func(c *gin.Context) {
		var l domain.Linea
		if err := c.ShouldBindJSON(&l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		l.Activo = true
		if l.Tipo == "" {
			l.Tipo = "Producción"
		}
		if l.Subtipo == "" {
			l.Subtipo = "Cámara"
		}
		if l.Mode == "" {
			l.Mode = "textil"
		}
		if !validModes[l.Mode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode debe ser 'textil'"})
			return
		}
		if err := repo.Create(c.Request.Context(), &l); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if eid := empresaIDFromContext(c); eid > 0 && notifier != nil {
			go notifier.Broadcast(c.Request.Context(), "plantas_lineas", eid, nil)
		}
		c.JSON(http.StatusCreated, l)
	})

	g.PUT("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		var l domain.Linea
		if err := c.ShouldBindJSON(&l); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		l.ID = id
		if l.Tipo == "" {
			l.Tipo = "Producción"
		}
		if l.Subtipo == "" {
			l.Subtipo = "Cámara"
		}
		if l.Mode == "" {
			l.Mode = "textil"
		}
		if !validModes[l.Mode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode debe ser 'textil'"})
			return
		}
		if err := repo.Update(c.Request.Context(), &l); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if eid := empresaIDFromContext(c); eid > 0 && notifier != nil {
			go notifier.Broadcast(c.Request.Context(), "plantas_lineas", eid, nil)
		}
		c.JSON(http.StatusOK, l)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if eid := empresaIDFromContext(c); eid > 0 && notifier != nil {
			go notifier.Broadcast(c.Request.Context(), "plantas_lineas", eid, nil)
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}

func seedDefaultParadaVariables(ctx context.Context, varRepo *repository.VariableRepo, d *domain.Dispositivo) {
	deviceID := d.ID
	types := []struct{ Clave, Nombre, Tipo string }{
		{"T_PARADA_PROGRAMADA", "Tiempo Parada Programada", "PARADA_PROGRAMADA"},
		{"T_PARADA_NO_PROGRAMADA", "Tiempo Parada No Programada", "PARADA_NO_PROGRAMADA"},
		{"T_REFRIGERIO", "Refrigerio", "PARADA_PROGRAMADA"},
		{"T_CAPACITACION_OBLIGATORIA", "Capacitación Obligatoria", "PARADA_PROGRAMADA"},
		{"T_MANTENIMIENTO_PLANIFICADO", "Mantenimiento Planificado", "PARADA_PROGRAMADA"},
		{"T_DISPONIBLE", "Tiempo Disponible", "OTRO"},
		{"MATERIAL", "Material", "OTRO"},
		{"DESTINO", "Destino", "OTRO"},
		{"MARCA", "Marca", "OTRO"},
		{"TAMANIO", "Tamaño", "OTRO"},
		{"CONTEO_UNITARIO_PRINCIPAL", "Conteo Unitario Principal", "OTRO"},
		{"T_MICROPARADA", "Tiempo de Microparada", "OTRO"},
		{"MERMA", "Merma", "OTRO"},
	}
	for _, t := range types {
		existing, _ := varRepo.FindByClave(ctx, t.Clave, deviceID)
		if existing != nil {
			continue
		}
		v := &domain.Variable{
			Nombre:        t.Nombre,
			Clave:         t.Clave,
			Valor:         "0",
			Tipo:          t.Tipo,
			DispositivoID: &deviceID,
			PlantaID:      d.PlantaID,
			EmpresaID:     d.EmpresaID,
			Activo:        true,
		}
		_ = varRepo.Create(ctx, v)
	}
}

// seedDerivedVariables crea automáticamente las variantes DELTA para variables
// que acumulan indefinidamente (conteo, merma). Llamar después de seedDefaultParadaVariables.
func seedDerivedVariables(ctx context.Context, varRepo *repository.VariableRepo, d *domain.Dispositivo) {
	deviceID := d.ID
	sources := []string{"CONTEO_UNITARIO_PRINCIPAL", "MERMA", "CONTEO_1"}
	for _, src := range sources {
		base, err := varRepo.FindByClave(ctx, src, deviceID)
		if err != nil || base == nil {
			continue
		}
		deltaClave := src + "_DELTA"
		existing, _ := varRepo.FindByClave(ctx, deltaClave, deviceID)
		if existing != nil {
			continue
		}
		v := &domain.Variable{
			Nombre:        base.Nombre + " (por intervalo)",
			Clave:         deltaClave,
			Valor:         "DELTA:" + src,
			Tipo:          "DERIVADA",
			DispositivoID: &deviceID,
			PlantaID:      d.PlantaID,
			EmpresaID:     d.EmpresaID,
			Activo:        true,
		}
		_ = varRepo.Create(ctx, v)
	}
}

func RegisterDispositivoRoutes(r *gin.RouterGroup, repo *repository.DispositivoRepo, provRepo *repository.DeviceProvisioningRepo, varRepo *repository.VariableRepo, mw *ports.JWTMiddleware) {
	g := r.Group("/dispositivos", mw.Auth())

	g.GET("", func(c *gin.Context) {
		dispositivos, err := provRepo.FindAllWithStatus(c.Request.Context(), resolveEmpresaID(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": dispositivos})
	})

	g.GET("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		d, err := provRepo.FindByIDWithStatus(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "dispositivo no encontrado"})
			return
		}
		c.JSON(http.StatusOK, d)
	})

	g.POST("", func(c *gin.Context) {
		var d domain.Dispositivo
		if err := c.ShouldBindJSON(&d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Enforce empresa_id from JWT
		if eid := resolveEmpresaID(c); eid != nil {
			d.EmpresaID = eid
		}
		apiKey, err := provRepo.Provision(c.Request.Context(), &d)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		seedDefaultParadaVariables(c.Request.Context(), varRepo, &d)
		seedDerivedVariables(c.Request.Context(), varRepo, &d)
		c.JSON(http.StatusCreated, gin.H{
			"dispositivo": d,
			"api_key":     apiKey,
		})
	})

	g.PUT("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		existing, err := provRepo.FindByIDWithStatus(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "dispositivo no encontrado"})
			return
		}
		// Ownership check
		if eid := resolveEmpresaID(c); eid != nil {
			if existing.EmpresaID == nil || *existing.EmpresaID != *eid {
				c.JSON(http.StatusForbidden, gin.H{"error": "acceso denegado"})
				return
			}
		}
		var d domain.Dispositivo
		if err := c.ShouldBindJSON(&d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		d.ID = id
		d.Activo = existing.Activo
		if eid := resolveEmpresaID(c); eid != nil {
			d.EmpresaID = eid
		}
		if err := provRepo.UpdateWithRegistry(c.Request.Context(), &d); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, d)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		existing, err := provRepo.FindByIDWithStatus(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "dispositivo no encontrado"})
			return
		}
		if eid := resolveEmpresaID(c); eid != nil {
			if existing.EmpresaID == nil || *existing.EmpresaID != *eid {
				c.JSON(http.StatusForbidden, gin.H{"error": "acceso denegado"})
				return
			}
		}
		if err := provRepo.Revoke(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	g.POST("/:id/regenerate-key", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		apiKey, err := provRepo.RegenerateKey(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"api_key": apiKey})
	})

	g.POST("/:id/activate", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		apiKey, err := provRepo.Activate(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"api_key": apiKey})
	})

	g.DELETE("/:id/permanent", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		existing, err := provRepo.FindByIDWithStatus(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "dispositivo no encontrado"})
			return
		}
		if eid := resolveEmpresaID(c); eid != nil {
			if existing.EmpresaID == nil || *existing.EmpresaID != *eid {
				c.JSON(http.StatusForbidden, gin.H{"error": "acceso denegado"})
				return
			}
		}
		if err := provRepo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}

func RegisterVariableRoutes(r *gin.RouterGroup, repo *repository.VariableRepo, dispRepo *repository.DispositivoRepo, mw *ports.JWTMiddleware, notifier *adapters.SyncNotifier) {
	g := r.Group("/variables", mw.Auth())

	// PATCH /api/variables/by-clave — actualiza el valor de una variable por
	// su clave y linea_id (para tablets en modo CLOUD sin dispositivo_id)
	g.PATCH("/by-clave", func(c *gin.Context) {
		var body struct {
			Clave   string `json:"clave"`
			Valor   string `json:"valor"`
			LineaID int    `json:"linea_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Clave == "" || body.LineaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "clave, valor y linea_id son requeridos"})
			return
		}
		dispositivos, err := dispRepo.FindByLinea(c.Request.Context(), body.LineaID)
		if err != nil || len(dispositivos) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "no se encontró dispositivo para la línea indicada"})
			return
		}
		dispositivoID := dispositivos[0].ID
		v, err := repo.FindByClave(c.Request.Context(), body.Clave, dispositivoID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "variable no encontrada"})
			return
		}
		v.Valor = body.Valor
		if err := repo.Update(c.Request.Context(), v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "clave": v.Clave, "valor": v.Valor})
	})

	g.GET("", func(c *gin.Context) {
		var dispID *int
		if v := c.Query("dispositivo_id"); v != "" {
			id, _ := strconv.Atoi(v)
			dispID = &id
		}
		variables, err := repo.FindAll(c.Request.Context(), dispID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": variables})
	})

	g.GET("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		v, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "variable no encontrada"})
			return
		}
		c.JSON(http.StatusOK, v)
	})

	syncVariables := func(c *gin.Context) {
		eid := empresaIDFromContext(c)
		if eid > 0 && notifier != nil {
			all, _ := repo.FindAll(c.Request.Context(), nil)
			go notifier.Broadcast(context.Background(), "variables", eid, all)
		}
	}

	g.POST("", func(c *gin.Context) {
		var v domain.Variable
		if err := c.ShouldBindJSON(&v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		v.Activo = true
		if err := repo.Create(c.Request.Context(), &v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		syncVariables(c)
		c.JSON(http.StatusCreated, v)
	})

	g.PUT("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		var v domain.Variable
		if err := c.ShouldBindJSON(&v); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		v.ID = id
		if err := repo.Update(c.Request.Context(), &v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		syncVariables(c)
		c.JSON(http.StatusOK, v)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		syncVariables(c)
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}

func RegisterLocacionRoutes(r *gin.RouterGroup, repo *repository.LocacionRepo, mw *ports.JWTMiddleware) {
	g := r.Group("/locaciones", mw.Auth())

	g.GET("", func(c *gin.Context) {
		var lineaID, empresaID *int
		if v := c.Query("linea_id"); v != "" {
			id, _ := strconv.Atoi(v)
			lineaID = &id
		}
		if v := c.Query("empresa_id"); v != "" {
			id, _ := strconv.Atoi(v)
			empresaID = &id
		}
		locaciones, err := repo.FindAll(c.Request.Context(), lineaID, empresaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": locaciones})
	})

	g.GET("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		loc, err := repo.FindByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "locación no encontrada"})
			return
		}
		c.JSON(http.StatusOK, loc)
	})

	g.POST("", func(c *gin.Context) {
		var loc domain.Locacion
		if err := c.ShouldBindJSON(&loc); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		loc.Activo = true
		if err := repo.Create(c.Request.Context(), &loc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, loc)
	})

	g.PUT("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		var loc domain.Locacion
		if err := c.ShouldBindJSON(&loc); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		loc.ID = id
		if err := repo.Update(c.Request.Context(), &loc); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, loc)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		id, _ := strconv.Atoi(c.Param("id"))
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}
