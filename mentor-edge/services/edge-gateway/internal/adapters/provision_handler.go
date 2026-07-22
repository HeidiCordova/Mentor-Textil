package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var validLineID = regexp.MustCompile(`^[0-9]+$`)

func (s *HTTPServer) handleProvisionLine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w)
		return
	}

	var req struct {
		LineaID      string `json:"linea_id"`
		CloudLineaID int    `json:"cloud_linea_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if !validLineID.MatchString(req.LineaID) {
		http.Error(w, `{"error":"linea_id must be a numeric string"}`, http.StatusBadRequest)
		return
	}

	schemaName := "linea_" + req.LineaID

	// Check if schema already exists
	var exists bool
	err := s.db.QueryRowContext(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`,
		schemaName,
	).Scan(&exists)
	if err != nil {
		s.internalError(w, fmt.Errorf("checking schema existence: %w", err))
		return
	}
	if exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "already_exists",
			"schema": schemaName,
		})
		return
	}

	// Read the SQL template from the mounted or bundled path
	templatePath := os.Getenv("LINEA_TEMPLATE_PATH")
	if templatePath == "" {
		templatePath = "/app/linea_template.sql"
	}
	templateSQL, err := os.ReadFile(templatePath)
	if err != nil {
		s.internalError(w, fmt.Errorf("reading template: %w", err))
		return
	}

	// Replace {schema} placeholder
	finalSQL := strings.ReplaceAll(string(templateSQL), "{schema}", schemaName)

	// Execute inside a transaction
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		s.internalError(w, fmt.Errorf("begin tx: %w", err))
		return
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(r.Context(), finalSQL); err != nil {
		s.internalError(w, fmt.Errorf("executing template for %s: %w", schemaName, err))
		return
	}

	// Ensure shared config schema exists and register line config.
	// NOTE: the config schema (CREATE TABLE config.line_config ...) is managed
	// exclusively by edge-config-service/postgres_repo.go#ensureConfigSchema,
	// which runs at startup. By the time /edge/provision is called from the UI
	// the config schema already exists.
	lineaIDInt := 0
	fmt.Sscanf(req.LineaID, "%d", &lineaIDInt)
	insertSQL := `INSERT INTO config.line_config (linea_id, device_id, cloud_linea_id)
		VALUES ($1,
		        COALESCE((SELECT device_id FROM config.line_config WHERE linea_id > 0 LIMIT 1), $2),
		        $3)
		ON CONFLICT (linea_id) DO NOTHING`
	if _, err := tx.ExecContext(r.Context(), insertSQL, lineaIDInt, req.LineaID, req.CloudLineaID); err != nil {
		log.Printf("[provision] warning: could not register line %s in config schema: %v", req.LineaID, err)
	}

	if err := tx.Commit(); err != nil {
		s.internalError(w, fmt.Errorf("commit: %w", err))
		return
	}

	log.Printf("[provision] schema %s created successfully", schemaName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "created",
		"schema": schemaName,
	})
}

func (s *HTTPServer) handleListSchemas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}

	rows, err := s.db.QueryContext(r.Context(),
		`SELECT schema_name FROM information_schema.schemata
		 WHERE schema_name LIKE 'linea_%' ORDER BY schema_name`)
	if err != nil {
		s.internalError(w, err)
		return
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			s.internalError(w, err)
			return
		}
		schemas = append(schemas, name)
	}
	if schemas == nil {
		schemas = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"schemas": schemas,
	})
}
