package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ensureMetersTable(ctx context.Context, db *pgxpool.Pool) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS energy.meters (
			id         SERIAL PRIMARY KEY,
			meter_id   VARCHAR(100) UNIQUE NOT NULL,
			unit_id    INT NOT NULL CHECK (unit_id BETWEEN 1 AND 247),
			ubicacion  TEXT,
			active     BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		log.Printf("ensureMetersTable DDL: %v", err)
		return
	}
	// Migraciones idempotentes de columnas
	type colMigration struct {
		name string
		ddl  string
	}
	for _, m := range []colMigration{
		{"ubicacion", `ALTER TABLE energy.meters ADD COLUMN ubicacion TEXT`},
		{"cloud_name", `ALTER TABLE energy.meters ADD COLUMN cloud_name TEXT`},
		{"cloud_token", `ALTER TABLE energy.meters ADD COLUMN cloud_token TEXT`},
	} {
		var exists bool
		_ = db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'energy' AND table_name = 'meters' AND column_name = $1
			)`, m.name).Scan(&exists)
		if !exists {
			if _, err = db.Exec(ctx, m.ddl); err != nil {
				log.Printf("ensureMetersTable alter %s: %v", m.name, err)
			}
		}
	}
	// Migracion idempotente: mover medidores desde energy.config si existen
	_, err = db.Exec(ctx, `
		DO $$
		BEGIN
			INSERT INTO energy.meters (meter_id, unit_id)
			SELECT c1.value,
			       COALESCE(NULLIF(c2.value, ''), '1')::INT
			FROM   energy.config c1
			JOIN   energy.config c2
			         ON c2.key = replace(c1.key, 'meter_id_', 'meter_unit_id_')
			WHERE  c1.key ~ '^meter_id_\d+$'
			  AND  c1.value != ''
			ON CONFLICT (meter_id) DO NOTHING;
		EXCEPTION
			WHEN OTHERS THEN NULL;
		END;
		$$`)
	if err != nil {
		log.Printf("ensureMetersTable migrate: %v", err)
	}
}

func registerWebHandlers(db *pgxpool.Pool, cfg *runtimeConfig) {
	go ensureMetersTable(context.Background(), db)

	http.HandleFunc("/api/meters/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodDelete:
			handleMeterDelete(w, r, db)
		case http.MethodPatch:
			handleMeterPatch(w, r, db)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/meters", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleMetersGet(w, r, db)
		case http.MethodPost:
			handleMeterCreate(w, r, db)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleConfigGet(w, r, db)
		case http.MethodPut:
			handleConfigPut(w, r, db, cfg)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handleStats(w, r, db)
	})

	http.HandleFunc("/api/snapshots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handleSnapshots(w, r, db)
	})

	http.HandleFunc("/api/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handleLatest(w, r, db)
	})

}

func handleConfigGet(w http.ResponseWriter, _ *http.Request, db *pgxpool.Pool) {
	rows, err := db.Query(context.Background(), `SELECT key, value, updated_at::text FROM energy.config ORDER BY key`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]string{}
	for rows.Next() {
		var k, v, ts string
		if err := rows.Scan(&k, &v, &ts); err != nil {
			continue
		}
		items = append(items, map[string]string{"key": k, "value": v, "updated_at": ts})
	}
	writeJSON(w, http.StatusOK, items)
}

func handleConfigPut(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool, cfg *runtimeConfig) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalido"})
		return
	}

	allowed := map[string]bool{
		"device_id": true, "cloud_url": true, "energy_api_key": true,
		"send_interval_s": true, "batch_size": true, "config_reload_s": true,
		"write_api_url": true, "command_poll_s": true,
	}

	updated := 0
	for k, v := range body {
		if !allowed[k] {
			continue
		}
		_, err := db.Exec(context.Background(),
			`INSERT INTO energy.config (key, value, updated_at) VALUES ($1, $2, NOW())
			 ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`, k, v)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		updated++
	}

	if err := cfg.load(context.Background(), db); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"updated": updated, "status": "ok"})
}

func handleStats(w http.ResponseWriter, _ *http.Request, db *pgxpool.Pool) {
	var total, synced, pending int64
	var lastSync, oldestPending *string

	db.QueryRow(context.Background(), `SELECT COUNT(*) FROM energy.snapshots`).Scan(&total)
	db.QueryRow(context.Background(), `SELECT COUNT(*) FROM energy.snapshots WHERE synced`).Scan(&synced)
	pending = total - synced
	db.QueryRow(context.Background(),
		`SELECT hora::text FROM energy.snapshots WHERE synced ORDER BY hora DESC LIMIT 1`).Scan(&lastSync)
	db.QueryRow(context.Background(),
		`SELECT hora::text FROM energy.snapshots WHERE NOT synced ORDER BY hora ASC LIMIT 1`).Scan(&oldestPending)

	writeJSON(w, http.StatusOK, map[string]any{
		"total":          total,
		"synced":         synced,
		"pending":        pending,
		"last_sync":      lastSync,
		"oldest_pending": oldestPending,
	})
}

func handleSnapshots(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	filter := r.URL.Query().Get("filter")
	where := ""
	switch filter {
	case "pending":
		where = "WHERE NOT synced"
	case "synced":
		where = "WHERE synced"
	}

	rows, err := db.Query(context.Background(), fmt.Sprintf(`
		SELECT id, device_id, meter_id, hora, interval_s, synced
		FROM energy.snapshots %s
		ORDER BY hora DESC LIMIT 50`, where))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var deviceID, meterID, hora string
		var intervalS int
		var syncedFlag bool
		if err := rows.Scan(&id, &deviceID, &meterID, &hora, &intervalS, &syncedFlag); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id": id, "device_id": deviceID, "meter_id": meterID,
			"hora": hora, "interval_s": intervalS, "synced": syncedFlag,
		})
	}
	writeJSON(w, http.StatusOK, items)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// nullIfEmpty convierte un puntero a string vacío en nil,
// para que COALESCE en SQL lo trate como "no modificar".
func nullIfEmpty(s **string) {
	if *s != nil && **s == "" {
		*s = nil
	}
}

// ─── Meters CRUD ─────────────────────────────────────────────────────────────

type meterRow struct {
	ID         int64   `json:"id"`
	MeterID    string  `json:"meter_id"`
	UnitID     int     `json:"unit_id"`
	Ubicacion  *string `json:"ubicacion,omitempty"`
	CloudName  *string `json:"cloud_name,omitempty"`
	CloudToken *string `json:"cloud_token,omitempty"`
}

func handleMetersGet(w http.ResponseWriter, _ *http.Request, db *pgxpool.Pool) {
	rows, err := db.Query(context.Background(),
		`SELECT id, meter_id, unit_id, ubicacion, cloud_name, cloud_token FROM energy.meters WHERE active ORDER BY id`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []meterRow{}
	for rows.Next() {
		var m meterRow
		if err := rows.Scan(&m.ID, &m.MeterID, &m.UnitID, &m.Ubicacion, &m.CloudName, &m.CloudToken); err != nil {
			continue
		}
		items = append(items, m)
	}
	writeJSON(w, http.StatusOK, items)
}

func handleMeterCreate(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	var body struct {
		MeterID    string  `json:"meter_id"`
		UnitID     int     `json:"unit_id"`
		Ubicacion  *string `json:"ubicacion"`
		CloudName  *string `json:"cloud_name"`
		CloudToken *string `json:"cloud_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalido"})
		return
	}
	if body.MeterID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "meter_id requerido"})
		return
	}
	if body.UnitID < 1 || body.UnitID > 247 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unit_id fuera de rango (1-247)"})
		return
	}
	if body.CloudToken == nil || strings.TrimSpace(*body.CloudToken) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cloud_token requerido"})
		return
	}
	nullIfEmpty(&body.CloudName)
	nullIfEmpty(&body.CloudToken)

	var id int64
	err := db.QueryRow(context.Background(),
		`INSERT INTO energy.meters (meter_id, unit_id, ubicacion, cloud_name, cloud_token)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (meter_id) DO UPDATE
		   SET unit_id    = $2,
		       ubicacion  = COALESCE($3, energy.meters.ubicacion),
		       cloud_name = COALESCE($4, energy.meters.cloud_name),
		       cloud_token = COALESCE($5, energy.meters.cloud_token),
		       active     = TRUE
		 RETURNING id`, body.MeterID, body.UnitID, body.Ubicacion, body.CloudName, body.CloudToken).Scan(&id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": "created"})
}

func handleMeterDelete(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/meters/")
	id, err := strconv.ParseInt(strings.TrimSuffix(idStr, "/"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id invalido"})
		return
	}
	ct, err := db.Exec(context.Background(),
		`UPDATE energy.meters SET active = FALSE WHERE id = $1`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "medidor no encontrado"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleMeterPatch(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/meters/")
	id, err := strconv.ParseInt(strings.TrimSuffix(idStr, "/"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id invalido"})
		return
	}
	var body struct {
		MeterID    *string `json:"meter_id"`
		UnitID     *int    `json:"unit_id"`
		Ubicacion  *string `json:"ubicacion"`
		CloudName  *string `json:"cloud_name"`
		CloudToken *string `json:"cloud_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "JSON invalido"})
		return
	}
	if body.UnitID != nil && (*body.UnitID < 1 || *body.UnitID > 247) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unit_id fuera de rango (1-247)"})
		return
	}
	nullIfEmpty(&body.CloudName)
	nullIfEmpty(&body.CloudToken)
	ct, err := db.Exec(context.Background(),
		`UPDATE energy.meters
		 SET meter_id    = COALESCE($1, meter_id),
		     unit_id     = COALESCE($2, unit_id),
		     ubicacion   = COALESCE($3, ubicacion),
		     cloud_name  = COALESCE($4, cloud_name),
		     cloud_token = COALESCE($5, cloud_token)
		 WHERE id = $6 AND active`, body.MeterID, body.UnitID, body.Ubicacion, body.CloudName, body.CloudToken, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if ct.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "medidor no encontrado"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func handleLatest(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	meterID := r.URL.Query().Get("meter_id")
	if meterID != "" {
		var id int64
		var deviceID, hora string
		var intervalS int
		var head, data []string
		err := db.QueryRow(context.Background(), `
			SELECT id, device_id, hora::text, interval_s, head, data
			FROM energy.snapshots
			WHERE meter_id = $1
			ORDER BY hora DESC LIMIT 1`, meterID).
			Scan(&id, &deviceID, &hora, &intervalS, &head, &data)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sin datos para ese medidor"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id": id, "device_id": deviceID, "meter_id": meterID,
			"hora": hora, "interval_s": intervalS, "head": head, "data": data,
		})
		return
	}

	rows, err := db.Query(context.Background(), `
		SELECT DISTINCT ON (meter_id)
			id, device_id, meter_id, hora::text, interval_s, head, data
		FROM energy.snapshots
		ORDER BY meter_id, hora DESC`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var deviceID, mID, hora string
		var intervalS int
		var head, data []string
		if err := rows.Scan(&id, &deviceID, &mID, &hora, &intervalS, &head, &data); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id": id, "device_id": deviceID, "meter_id": mID,
			"hora": hora, "interval_s": intervalS, "head": head, "data": data,
		})
	}
	writeJSON(w, http.StatusOK, items)
}
