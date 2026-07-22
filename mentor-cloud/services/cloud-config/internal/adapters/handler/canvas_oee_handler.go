package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"cloud-config/internal/ports"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultOEEGraph contiene el grafo OEE estándar del sistema en formato Vue Flow.
const defaultOEEGraph = `{
  "nodes": [
    {"id":"v_tdt",      "type":"variableNode", "position":{"x":30,  "y":20},  "data":{"label":"T_DISPONIBLE_TOTAL",      "clave":"T_DISPONIBLE_TOTAL",      "fuente":"turno",    "descripcion":"Tiempo Disponible Total (viene de Turnos)"}},
    {"id":"v_tpp",      "type":"variableNode", "position":{"x":30,  "y":130}, "data":{"label":"T_PARADA_PROGRAMADA",     "clave":"T_PARADA_PROGRAMADA",     "fuente":"variable", "descripcion":"Tiempo de Parada Programada"}},
    {"id":"v_tpnp",     "type":"variableNode", "position":{"x":30,  "y":280}, "data":{"label":"T_PARADA_NO_PROGRAMADA",  "clave":"T_PARADA_NO_PROGRAMADA",  "fuente":"variable", "descripcion":"Tiempo de Parada No Programada"}},
    {"id":"v_mp",       "type":"variableNode", "position":{"x":30,  "y":420}, "data":{"label":"T_MICROPARADA",           "clave":"T_MICROPARADA",           "fuente":"variable", "descripcion":"Tiempo de Microparada"}},
    {"id":"v_prod",     "type":"variableNode", "position":{"x":30,  "y":560}, "data":{"label":"PRODUCCION",              "clave":"CONTEO_UNITARIO_PRINCIPAL","fuente":"variable", "descripcion":"Producción (conteo unitario principal)"}},
    {"id":"v_vn",       "type":"variableNode", "position":{"x":30,  "y":700}, "data":{"label":"VELOCIDAD_NOMINAL",       "clave":"VELOCIDAD_NOMINAL",        "fuente":"config",   "descripcion":"Velocidad Nominal (configuración de línea)"}},
    {"id":"v_merma",    "type":"variableNode", "position":{"x":30,  "y":840}, "data":{"label":"MERMA",                   "clave":"MERMA",                    "fuente":"variable", "descripcion":"Merma"}},

    {"id":"f_td",       "type":"formulaNode",  "position":{"x":360, "y":60},  "data":{"label":"T_DISPONIBLE",    "formula":"a − b",   "operator":"-", "inputLabels":["T_DISP_TOTAL","T_PARADA_PROG"],    "descripcion":"Tiempo Disponible = T_DIS_TOTAL − T_PARADA_PROG"}},
    {"id":"f_to",       "type":"formulaNode",  "position":{"x":360, "y":250}, "data":{"label":"T_OPERATIVO",     "formula":"a − b",   "operator":"-", "inputLabels":["T_DISPONIBLE","T_PARADA_NO_PROG"], "descripcion":"Tiempo Operativo = T_DISPONIBLE − T_PARADA_NO_PROG"}},
    {"id":"f_tnp",      "type":"formulaNode",  "position":{"x":680, "y":360}, "data":{"label":"T_NETO_PROD",     "formula":"a − b",   "operator":"-", "inputLabels":["T_OPERATIVO","T_MICROPARADA"],      "descripcion":"Tiempo Neto de Producción = T_OPERATIVO − T_MICROPARADA"}},
    {"id":"f_tnomprod", "type":"formulaNode",  "position":{"x":680, "y":560}, "data":{"label":"T_NOMINAL_PROD",  "formula":"a ÷ b",   "operator":"/", "inputLabels":["PRODUCCION","VELOCIDAD_NOMINAL"],   "descripcion":"Tiempo Nominal de Producción = PRODUCCION ÷ VELOCIDAD_NOMINAL"}},
    {"id":"f_tpv",      "type":"formulaNode",  "position":{"x":980, "y":450}, "data":{"label":"T_PERDIDO_VEL",   "formula":"a − b",   "operator":"-", "inputLabels":["T_NETO_PROD","T_NOMINAL_PROD"],     "descripcion":"Tiempo Perdido por Baja Velocidad = T_NETO − T_NOMINAL"}},

    {"id":"k_disp",     "type":"kpiNode",      "position":{"x":680, "y":40},  "data":{"label":"DISPONIBILIDAD",  "formula":"T_OP ÷ T_DIS",                          "color":"#3b82f6", "descripcion":"Disponibilidad = T_OPERATIVO / T_DISPONIBLE"}},
    {"id":"k_rend",     "type":"kpiNode",      "position":{"x":1260,"y":300}, "data":{"label":"RENDIMIENTO",     "formula":"(T_OP − T_MP − T_PV) ÷ T_OP",           "color":"#22c55e", "descripcion":"Rendimiento = (T_OP − T_MP − T_PV) / T_OP"}},
    {"id":"k_cal",      "type":"kpiNode",      "position":{"x":1260,"y":620}, "data":{"label":"CALIDAD",         "formula":"(PROD − MERMA) ÷ PROD",                  "color":"#f59e0b", "descripcion":"Calidad = (PRODUCCION − MERMA) / PRODUCCION"}},
    {"id":"k_oee",      "type":"oeeNode",      "position":{"x":1560,"y":440}, "data":{"label":"OEE",             "formula":"DISPONIBILIDAD × RENDIMIENTO × CALIDAD", "color":"#ef4444", "descripcion":"OEE = DISPONIBILIDAD × RENDIMIENTO × CALIDAD"}}
  ],
  "edges": [
    {"id":"e1",  "source":"v_tdt",      "target":"f_td",       "targetHandle":"a", "animated":true},
    {"id":"e2",  "source":"v_tpp",      "target":"f_td",       "targetHandle":"b", "animated":true},
    {"id":"e3",  "source":"f_td",       "target":"f_to",       "targetHandle":"a", "animated":true},
    {"id":"e4",  "source":"v_tpnp",     "target":"f_to",       "targetHandle":"b", "animated":true},
    {"id":"e5",  "source":"f_to",       "target":"k_disp",     "targetHandle":"a", "animated":true, "label":"T_OP"},
    {"id":"e6",  "source":"f_td",       "target":"k_disp",     "targetHandle":"b", "animated":true, "label":"T_DIS"},
    {"id":"e7",  "source":"f_to",       "target":"f_tnp",      "targetHandle":"a", "animated":true},
    {"id":"e8",  "source":"v_mp",       "target":"f_tnp",      "targetHandle":"b", "animated":true},
    {"id":"e9",  "source":"v_prod",     "target":"f_tnomprod", "targetHandle":"a", "animated":true},
    {"id":"e10", "source":"v_vn",       "target":"f_tnomprod", "targetHandle":"b", "animated":true},
    {"id":"e11", "source":"f_tnp",      "target":"f_tpv",      "targetHandle":"a", "animated":true},
    {"id":"e12", "source":"f_tnomprod", "target":"f_tpv",      "targetHandle":"b", "animated":true},
    {"id":"e13", "source":"f_to",       "target":"k_rend",     "targetHandle":"a", "animated":true, "label":"T_OP"},
    {"id":"e14", "source":"v_mp",       "target":"k_rend",     "targetHandle":"b", "animated":true, "label":"T_MP"},
    {"id":"e15", "source":"f_tpv",      "target":"k_rend",     "targetHandle":"c", "animated":true, "label":"T_PV"},
    {"id":"e16", "source":"v_prod",     "target":"k_cal",      "targetHandle":"a", "animated":true},
    {"id":"e17", "source":"v_merma",    "target":"k_cal",      "targetHandle":"b", "animated":true},
    {"id":"e18", "source":"k_disp",     "target":"k_oee",      "targetHandle":"a", "animated":true, "style":{"stroke":"#3b82f6","strokeWidth":2}},
    {"id":"e19", "source":"k_rend",     "target":"k_oee",      "targetHandle":"b", "animated":true, "style":{"stroke":"#22c55e","strokeWidth":2}},
    {"id":"e20", "source":"k_cal",      "target":"k_oee",      "targetHandle":"c", "animated":true, "style":{"stroke":"#f59e0b","strokeWidth":2}}
  ]
}`

// RegisterCanvasOEERoutes registra los endpoints REST del canvas OEE.
// La resolución sigue la jerarquía: dispositivo -> planta -> system.
func RegisterCanvasOEERoutes(r *gin.RouterGroup, db *pgxpool.Pool, mw *ports.JWTMiddleware) {
	g := r.Group("/canvas-oee", mw.Auth())

	g.GET("", func(c *gin.Context) {
		ctx := c.Request.Context()

		dispositivoID := c.Query("dispositivo_id")
		plantaID := c.Query("planta_id")

		if dispositivoID != "" {
			row, found, err := getCanvas(ctx, db, "dispositivo", nil, &dispositivoID)
			if err == nil && found {
				c.JSON(http.StatusOK, row)
				return
			}
		}
		if plantaID != "" {
			row, found, err := getCanvas(ctx, db, "planta", &plantaID, nil)
			if err == nil && found {
				c.JSON(http.StatusOK, row)
				return
			}
		}
		row, found, err := getCanvas(ctx, db, "system", nil, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !found {
			if err2 := ensureSystemDefault(ctx, db); err2 != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear el default del sistema"})
				return
			}
			row, _, _ = getCanvas(ctx, db, "system", nil, nil)
		}
		c.JSON(http.StatusOK, row)
	})

	g.PUT("", func(c *gin.Context) {
		var body struct {
			Scope         string          `json:"scope"`
			PlantaID      *int            `json:"planta_id"`
			DispositivoID *int            `json:"dispositivo_id"`
			Nombre        string          `json:"nombre"`
			Grafo         json.RawMessage `json:"grafo"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Scope == "" {
			body.Scope = "system"
		}
		if body.Nombre == "" {
			body.Nombre = "Fórmula OEE"
		}
		if len(body.Grafo) == 0 {
			body.Grafo = json.RawMessage(`{"nodes":[],"edges":[]}`)
		}

		ctx := c.Request.Context()
		grafoJSON, _ := json.Marshal(body.Grafo)

		_, err := db.Exec(ctx, `
			INSERT INTO config.canvas_oee (scope, planta_id, dispositivo_id, nombre, grafo, updated_at)
			VALUES ($1, $2, $3, $4, $5::jsonb, now())
			ON CONFLICT ON CONSTRAINT canvas_oee_unique_scope
			DO UPDATE SET nombre=$4, grafo=$5::jsonb, updated_at=now()
		`, body.Scope, body.PlantaID, body.DispositivoID, body.Nombre, string(grafoJSON))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	g.POST("/reset-default", func(c *gin.Context) {
		var body struct {
			Scope         string `json:"scope"`
			PlantaID      *int   `json:"planta_id"`
			DispositivoID *int   `json:"dispositivo_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil && err != io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
			return
		}
		if body.Scope == "" {
			body.Scope = "system"
		}

		ctx := c.Request.Context()
		_, err := db.Exec(ctx, `
			INSERT INTO config.canvas_oee (scope, planta_id, dispositivo_id, nombre, grafo, updated_at)
			VALUES ($1, $2, $3, 'Fórmula OEE', $4::jsonb, now())
			ON CONFLICT ON CONSTRAINT canvas_oee_unique_scope
			DO UPDATE SET nombre='Fórmula OEE', grafo=$4::jsonb, updated_at=now()
		`, body.Scope, body.PlantaID, body.DispositivoID, defaultOEEGraph)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Canvas restablecido al default OEE estándar"})
	})
}

type canvasOEERow struct {
	ID            int             `json:"id"`
	Scope         string          `json:"scope"`
	PlantaID      *int            `json:"planta_id"`
	DispositivoID *int            `json:"dispositivo_id"`
	Nombre        string          `json:"nombre"`
	Grafo         json.RawMessage `json:"grafo"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func getCanvas(ctx context.Context, db *pgxpool.Pool, scope string, plantaID *string, dispositivoID *string) (*canvasOEERow, bool, error) {
	var row canvasOEERow
	var grafoBytes []byte

	var q string
	var args []interface{}

	switch scope {
	case "dispositivo":
		did, _ := strconv.Atoi(*dispositivoID)
		q = `SELECT id, scope, planta_id, dispositivo_id, nombre, grafo, updated_at
			 FROM config.canvas_oee WHERE scope='dispositivo' AND dispositivo_id=$1 LIMIT 1`
		args = []interface{}{did}
	case "planta":
		pid, _ := strconv.Atoi(*plantaID)
		q = `SELECT id, scope, planta_id, dispositivo_id, nombre, grafo, updated_at
			 FROM config.canvas_oee WHERE scope='planta' AND planta_id=$1 LIMIT 1`
		args = []interface{}{pid}
	default:
		q = `SELECT id, scope, planta_id, dispositivo_id, nombre, grafo, updated_at
			 FROM config.canvas_oee WHERE scope='system' LIMIT 1`
		args = nil
	}

	err := db.QueryRow(ctx, q, args...).Scan(
		&row.ID, &row.Scope, &row.PlantaID, &row.DispositivoID,
		&row.Nombre, &grafoBytes, &row.UpdatedAt,
	)
	if err != nil {
		return nil, false, err
	}
	row.Grafo = json.RawMessage(grafoBytes)
	return &row, true, nil
}

func ensureSystemDefault(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		INSERT INTO config.canvas_oee (scope, planta_id, dispositivo_id, nombre, grafo)
		VALUES ('system', NULL, NULL, 'Fórmula OEE', $1::jsonb)
		ON CONFLICT ON CONSTRAINT canvas_oee_unique_scope DO NOTHING
	`, defaultOEEGraph)
	return err
}
