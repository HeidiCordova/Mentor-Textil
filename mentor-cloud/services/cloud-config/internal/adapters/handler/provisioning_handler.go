package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud-config/internal/ports"

	"mentor.local/shared/multitenancy"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProvisioningHandler struct {
	plantaRepo     ports.PlantaRepository
	lineaRepo      ports.LineaRepository
	masterPool     *pgxpool.Pool
	provisioner    multitenancy.DBProvisioner
	credentials    *multitenancy.AESCredentialProvider
	energyTemplSQL string
}

func NewProvisioningHandler(
	plantaRepo ports.PlantaRepository,
	lineaRepo ports.LineaRepository,
	masterPool *pgxpool.Pool,
	provisioner multitenancy.DBProvisioner,
	credentials *multitenancy.AESCredentialProvider,
	energyTemplSQL string,
) *ProvisioningHandler {
	return &ProvisioningHandler{
		plantaRepo:     plantaRepo,
		lineaRepo:      lineaRepo,
		masterPool:     masterPool,
		provisioner:    provisioner,
		credentials:    credentials,
		energyTemplSQL: energyTemplSQL,
	}
}

func RegisterProvisioningRoutes(
	rg *gin.RouterGroup,
	h *ProvisioningHandler,
	mw *ports.JWTMiddleware,
) {
	admin := rg.Group("/admin")
	admin.Use(mw.Auth(), requireRole("superadmin"))
	admin.POST("/plantas/:id/provision", h.Provision)
	admin.POST("/plantas/:id/lineas/:lineaId/provision", h.ProvisionLinea)
}

func requireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, _ := c.Get("user_role")
		if r != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acceso denegado"})
			return
		}
		c.Next()
	}
}

func (h *ProvisioningHandler) Provision(c *gin.Context) {
	plantaID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "planta_id invalido"})
		return
	}

	ctx := c.Request.Context()

	planta, err := h.plantaRepo.FindByID(ctx, plantaID)
	if err != nil || planta == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "planta no encontrada"})
		return
	}

	var exists bool
	err = h.masterPool.QueryRow(ctx,
		`SELECT provisioned FROM admin.planta_databases WHERE planta_id = $1`, plantaID,
	).Scan(&exists)
	if err == nil && exists {
		c.JSON(http.StatusConflict, gin.H{"error": "planta ya provisionada"})
		return
	}

	dbName := fmt.Sprintf("mentor_planta_%d", plantaID)
	pgUser := fmt.Sprintf("planta_%d_user", plantaID)
	pgPassword := generatePassword()

	if err := h.provisioner.CreateDatabase(ctx, dbName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "crear base de datos: " + err.Error()})
		return
	}
	if err := h.provisioner.CreateUser(ctx, pgUser, pgPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "crear usuario: " + err.Error()})
		return
	}
	if err := h.provisioner.GrantAccess(ctx, pgUser, dbName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "grant access: " + err.Error()})
		return
	}

	masterConnCfg := h.masterPool.Config().ConnConfig
	dbHost := masterConnCfg.Host
	dbPort := masterConnCfg.Port // uint16

	lineas, err := h.lineaRepo.FindAll(ctx, &plantaID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "listar lineas: " + err.Error()})
		return
	}

	// Inicializar como slices vacíos (no nil) para evitar NULL en columnas NOT NULL de postgres
	schemasCreados := []string{}
	lineaIDs := []int{}

	if len(lineas) > 0 {
		// Conectar a la nueva BD con las credenciales del master (tiene privilegios para crear schemas)
		provisionDSN := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
			dbHost, dbPort, dbName, masterConnCfg.User, masterConnCfg.Password)

		plantaPool, err := pgxpool.New(ctx, provisionDSN)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "conectar a nueva BD: " + err.Error()})
			return
		}
		defer plantaPool.Close()

		for _, l := range lineas {
			schema := multitenancy.SchemaName(l.ID)
			if err := h.provisioner.CreateSchema(ctx, plantaPool, schema); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("crear schema %s: %v", schema, err)})
				return
			}
			if l.Tipo == "Energía" {
				if h.energyTemplSQL != "" {
					sql := strings.ReplaceAll(h.energyTemplSQL, "{schema}", schema)
					if _, err := plantaPool.Exec(ctx, sql); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("energy template %s: %v", schema, err)})
						return
					}
				}
			} else {
				if err := h.provisioner.RunTemplate(ctx, plantaPool, schema); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("template %s: %v", schema, err)})
					return
				}
			}
			// Dar permisos al usuario de la planta sobre el nuevo schema
			grantSQL := fmt.Sprintf(
				`GRANT USAGE ON SCHEMA %s TO %s;
				 GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %s TO %s;
				 GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA %s TO %s;
				 ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s;
				 ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO %s`,
				schema, pgUser, schema, pgUser, schema, pgUser, schema, pgUser, schema, pgUser,
			)
			if _, err := plantaPool.Exec(ctx, grantSQL); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("grant schema %s: %v", schema, err)})
				return
			}
			schemasCreados = append(schemasCreados, schema)
			lineaIDs = append(lineaIDs, l.ID)
		}
	}

	encPassword, err := h.credentials.Encrypt(pgPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt password: " + err.Error()})
		return
	}

	_, err = h.masterPool.Exec(ctx, `
		INSERT INTO admin.planta_databases
			(planta_id, db_name, pg_user, pg_password_enc, host, port, instance_type, provisioned, provisioned_at, lineas_creadas)
		VALUES ($1, $2, $3, $4, $7, $8, 'shared', true, $5, $6)
		ON CONFLICT (planta_id) DO UPDATE SET
			db_name = EXCLUDED.db_name, pg_user = EXCLUDED.pg_user,
			pg_password_enc = EXCLUDED.pg_password_enc,
			provisioned = true, provisioned_at = EXCLUDED.provisioned_at,
			lineas_creadas = EXCLUDED.lineas_creadas`,
		plantaID, dbName, pgUser, encPassword, time.Now(), lineaIDs, dbHost, int(dbPort),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "registrar provisioning: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"planta_id": plantaID,
		"db_name":   dbName,
		"schemas":   schemasCreados,
		"status":    "provisioned",
	})
}

func generatePassword() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ProvisionLinea crea el schema linea_X en la BD de la planta y ejecuta el template SQL.
// Requiere que la planta ya esté provisionada.
func (h *ProvisioningHandler) ProvisionLinea(c *gin.Context) {
	plantaID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "planta_id invalido"})
		return
	}
	lineaID, err := strconv.Atoi(c.Param("lineaId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id invalido"})
		return
	}

	ctx := c.Request.Context()

	// Verificar que la planta está provisionada y obtener credenciales
	var entry struct {
		DBName        string
		PGUser        string
		PGPasswordEnc string
		Host          string
		Port          int
		LineasCreadas []int
	}
	err = h.masterPool.QueryRow(ctx,
		`SELECT db_name, pg_user, pg_password_enc, host, port,
		        COALESCE(lineas_creadas, '{}')
		 FROM admin.planta_databases
		 WHERE planta_id = $1 AND provisioned = true`, plantaID,
	).Scan(&entry.DBName, &entry.PGUser, &entry.PGPasswordEnc,
		&entry.Host, &entry.Port, &entry.LineasCreadas)
	if err != nil {
		c.JSON(http.StatusPreconditionFailed, gin.H{"error": "planta no provisionada — ejecute primero POST /admin/plantas/:id/provision"})
		return
	}

	// Verificar que la línea existe y pertenece a la planta
	linea, err := h.lineaRepo.FindByID(ctx, lineaID)
	if err != nil || linea == nil || linea.PlantaID != plantaID {
		c.JSON(http.StatusNotFound, gin.H{"error": "linea no encontrada en esta planta"})
		return
	}

	schema := multitenancy.SchemaName(lineaID)

	// Verificar si ya está creado
	for _, id := range entry.LineasCreadas {
		if id == lineaID {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("schema %s ya existe", schema)})
			return
		}
	}

	// Conectar a la BD de la planta (usando credenciales del master para provisionar)
	masterCfg := h.masterPool.Config().ConnConfig
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		entry.Host, entry.Port, entry.DBName, masterCfg.User, masterCfg.Password)

	plantaPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "conectar BD planta: " + err.Error()})
		return
	}
	defer plantaPool.Close()

	if err := h.provisioner.CreateSchema(ctx, plantaPool, schema); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "crear schema: " + err.Error()})
		return
	}
	if linea.Tipo == "Energía" {
		if h.energyTemplSQL != "" {
			sql := strings.ReplaceAll(h.energyTemplSQL, "{schema}", schema)
			if _, err := plantaPool.Exec(ctx, sql); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "energy template: " + err.Error()})
				return
			}
		}
	} else {
		if err := h.provisioner.RunTemplate(ctx, plantaPool, schema); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "ejecutar template: " + err.Error()})
			return
		}
	}
	// Dar permisos al usuario de la planta sobre el nuevo schema
	grantSQL := fmt.Sprintf(
		`GRANT USAGE ON SCHEMA %s TO %s;
		 GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %s TO %s;
		 GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA %s TO %s;
		 ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s;
		 ALTER DEFAULT PRIVILEGES IN SCHEMA %s GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO %s`,
		schema, entry.PGUser, schema, entry.PGUser, schema, entry.PGUser, schema, entry.PGUser, schema, entry.PGUser,
	)
	if _, err := plantaPool.Exec(ctx, grantSQL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "grant schema: " + err.Error()})
		return
	}

	// Actualizar lineas_creadas en admin.planta_databases
	newLineas := append(entry.LineasCreadas, lineaID)
	_, err = h.masterPool.Exec(ctx,
		`UPDATE admin.planta_databases SET lineas_creadas = $1 WHERE planta_id = $2`,
		newLineas, plantaID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "actualizar registro: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"planta_id": plantaID,
		"linea_id":  lineaID,
		"schema":    schema,
		"status":    "provisioned",
	})
}
