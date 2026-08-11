package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"edge-config-service/internal/domain"
	"edge-config-service/internal/ports"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(connStr string) (ports.ConfigStorage, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := ensureConfigSchema(db); err != nil {
		return nil, fmt.Errorf("ensure config schema: %w", err)
	}

	return &PostgresRepo{db: db}, nil
}

// ensureConfigSchema creates the shared config schema and line_config table.
// linea_id is the UNIQUE key: 0=system defaults, 1..N=production lines.
// device_id is NOT unique — multiple lines share the same Jetson device_id.
func ensureConfigSchema(db *sql.DB) error {
	ddl := `
		CREATE SCHEMA IF NOT EXISTS config;

		CREATE TABLE IF NOT EXISTS config.line_config (
			id              SERIAL PRIMARY KEY,
			device_id       VARCHAR(100) NOT NULL DEFAULT '',
			config_version  INTEGER NOT NULL DEFAULT 1,
			linea_id        INTEGER NOT NULL DEFAULT 0,
			empresa_id      INTEGER NOT NULL DEFAULT 0,
			cloud_linea_id  INTEGER NOT NULL DEFAULT 0,
			roi             JSONB NOT NULL DEFAULT '{"x":120,"y":60,"width":320,"height":200}',
			thresholds      JSONB NOT NULL DEFAULT '{"edge":0.4,"color":0.6,"flow":0.5,"dy":5.0,"beige":0.35,"high":0.7,"low":0.3}',
			fsm             JSONB NOT NULL DEFAULT '{"n_frames":3,"cooldown":8,"exit_frames":5,"max_wait_exit_frames":50}',
			mode            VARCHAR(30) NOT NULL DEFAULT 'textil',
			camera          JSONB,
			oee             JSONB NOT NULL DEFAULT '{"line_name":"","micro_stop_max_s":120,"stop_max_s":86400,"snapshot_interval_s":1800,"vel_unit":"uh","vel_nominal_us":0.008333333}',
			cloud           JSONB NOT NULL DEFAULT '{"sync_interval_s":300}',
			tablet          JSONB,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT chk_line_config_mode_textil_only CHECK (mode = 'textil')
		);

		-- Ensure columns exist on pre-existing tables
		ALTER TABLE config.line_config ADD COLUMN IF NOT EXISTS linea_id INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE config.line_config ADD COLUMN IF NOT EXISTS empresa_id INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE config.line_config ADD COLUMN IF NOT EXISTS cloud_linea_id INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE config.line_config ADD COLUMN IF NOT EXISTS roi_presencia JSONB;
		ALTER TABLE config.line_config ALTER COLUMN mode SET DEFAULT 'textil';
		ALTER TABLE config.line_config ALTER COLUMN oee
			SET DEFAULT '{"line_name":"","micro_stop_max_s":120,"stop_max_s":86400,"snapshot_interval_s":1800,"vel_unit":"uh","vel_nominal_us":0.008333333}'::JSONB;

		-- Migration: linea_id becomes the unique key instead of device_id
		ALTER TABLE config.line_config DROP CONSTRAINT IF EXISTS line_config_device_id_key;

		-- Clean duplicate linea_id=0 rows (test data) keeping only _system
		DELETE FROM config.line_config
			WHERE linea_id = 0 AND device_id != '_system'
			  AND id NOT IN (SELECT MIN(id) FROM config.line_config WHERE device_id = '_system');

		-- Ensure _system row exists (linea_id=0)
		INSERT INTO config.line_config (device_id, linea_id)
			VALUES ('_system', 0)
			ON CONFLICT DO NOTHING;

		CREATE OR REPLACE FUNCTION config.bump_config_version()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.config_version := OLD.config_version + 1;
			NEW.updated_at := NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS trg_config_bump_version ON config.line_config;
		CREATE TRIGGER trg_config_bump_version
		BEFORE UPDATE ON config.line_config
		FOR EACH ROW
		EXECUTE FUNCTION config.bump_config_version();
	`
	if _, err := db.Exec(ddl); err != nil {
		return err
	}

	// Add UNIQUE index on linea_id (idempotent). Done separately because
	// existing tables may have duplicate linea_id=0 rows that the DELETE above cleaned.
	_, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uidx_linea_id ON config.line_config (linea_id)`)
	if err != nil {
		log.Printf("[config] warning: could not create unique index on linea_id: %v", err)
		// Non-fatal: may need manual cleanup if old rows remain
	}
	return nil
}

func (r *PostgresRepo) tbl() string { return "config.line_config" }

// GetConfig retrieves config by linea_id. linea_id=0 returns system defaults.
func (r *PostgresRepo) GetConfig(ctx context.Context, lineaID int) (*domain.LineConfig, error) {
	query := fmt.Sprintf(`
		SELECT id, device_id, config_version, linea_id, empresa_id, roi, roi_presencia, thresholds, fsm, mode, camera, oee, cloud, tablet, created_at, updated_at
		FROM %s
		WHERE linea_id = $1
	`, r.tbl())

	config := &domain.LineConfig{}
	var roiJSON, roiPresenciaJSON, thresholdsJSON, fsmJSON, cameraJSON, oeeJSON, cloudJSON []byte
	var tabletJSON []byte

	err := r.db.QueryRowContext(ctx, query, lineaID).Scan(
		&config.ID,
		&config.DeviceID,
		&config.ConfigVersion,
		&config.LineaID,
		&config.EmpresaID,
		&roiJSON,
		&roiPresenciaJSON,
		&thresholdsJSON,
		&fsmJSON,
		&config.Mode,
		&cameraJSON,
		&oeeJSON,
		&cloudJSON,
		&tabletJSON,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrConfigNotFound
	}

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(roiJSON, &config.ROI); err != nil {
		return nil, err
	}

	if roiPresenciaJSON != nil {
		var roiP domain.ROIData
		if err := json.Unmarshal(roiPresenciaJSON, &roiP); err == nil {
			config.ROIPresencia = &roiP
		}
	}

	if err := json.Unmarshal(thresholdsJSON, &config.Thresholds); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(fsmJSON, &config.FSM); err != nil {
		return nil, err
	}

	if cameraJSON != nil {
		var camera domain.Camera
		if err := json.Unmarshal(cameraJSON, &camera); err == nil {
			config.Camera = &camera
		}
	}

	if oeeJSON != nil {
		if err := json.Unmarshal(oeeJSON, &config.OEE); err != nil {
			return nil, err
		}
	}

	if cloudJSON != nil {
		if err := json.Unmarshal(cloudJSON, &config.Cloud); err != nil {
			return nil, err
		}
	}

	if tabletJSON != nil {
		var tablet domain.TabletConfig
		if err := json.Unmarshal(tabletJSON, &tablet); err == nil {
			config.Tablet = &tablet
		}
	}

	return config, nil
}

// UpdateConfig upserts by linea_id (0 for system, >=1 for lines).
func (r *PostgresRepo) UpdateConfig(ctx context.Context, config *domain.LineConfig) error {
	roiJSON, _ := json.Marshal(config.ROI)
	thresholdsJSON, _ := json.Marshal(config.Thresholds)
	fsmJSON, _ := json.Marshal(config.FSM)
	oeeJSON, _ := json.Marshal(config.OEE)
	cloudJSON, _ := json.Marshal(config.Cloud)

	var roiPresenciaJSON []byte
	if config.ROIPresencia != nil {
		roiPresenciaJSON, _ = json.Marshal(config.ROIPresencia)
	}

	var cameraJSON []byte
	if config.Camera != nil {
		cameraJSON, _ = json.Marshal(config.Camera)
	}

	var tabletJSON []byte
	if config.Tablet != nil {
		tabletJSON, _ = json.Marshal(config.Tablet)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (linea_id, device_id, empresa_id, roi, roi_presencia, thresholds, fsm, mode, camera, oee, cloud, tablet)
		VALUES ($9, $11, $10, $1, $12, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (linea_id) DO UPDATE
			SET device_id = EXCLUDED.device_id,
			    empresa_id = EXCLUDED.empresa_id,
			    roi = EXCLUDED.roi,
			    roi_presencia = EXCLUDED.roi_presencia,
			    thresholds = EXCLUDED.thresholds,
			    fsm = EXCLUDED.fsm,
			    mode = EXCLUDED.mode,
			    camera = EXCLUDED.camera,
			    oee = EXCLUDED.oee,
			    cloud = EXCLUDED.cloud,
			    tablet = EXCLUDED.tablet,
			    updated_at = NOW()
	`, r.tbl())

	_, err := r.db.ExecContext(ctx, query,
		roiJSON,          // $1
		thresholdsJSON,   // $2
		fsmJSON,          // $3
		config.Mode,      // $4
		cameraJSON,       // $5
		oeeJSON,          // $6
		cloudJSON,        // $7
		tabletJSON,       // $8
		config.LineaID,   // $9
		config.EmpresaID, // $10
		config.DeviceID,  // $11
		roiPresenciaJSON, // $12
	)

	return err
}

// GetLineIDs returns all linea_id values for real lines (>0), sorted.
func (r *PostgresRepo) GetLineIDs(ctx context.Context) ([]int, error) {
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT linea_id FROM %s WHERE linea_id > 0 ORDER BY linea_id`, r.tbl()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func (r *PostgresRepo) GetConfigVersion(ctx context.Context, lineaID int) (int, error) {
	query := fmt.Sprintf(`SELECT config_version FROM %s WHERE linea_id = $1`, r.tbl())

	var version int
	err := r.db.QueryRowContext(ctx, query, lineaID).Scan(&version)
	if err == sql.ErrNoRows {
		return 0, domain.ErrConfigNotFound
	}

	return version, err
}

func (r *PostgresRepo) DeleteLine(ctx context.Context, lineaID int) error {
	if lineaID <= 0 {
		return fmt.Errorf("cannot delete system config (linea_id=0)")
	}
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE linea_id = $1`, r.tbl()), lineaID)
	return err
}

// GetDeviceID returns the Jetson's device_id from any real line config row.
func (r *PostgresRepo) GetDeviceID(ctx context.Context) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT device_id FROM %s WHERE linea_id > 0 ORDER BY linea_id LIMIT 1`, r.tbl()),
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// SetDeviceID updates the device_id on ALL real line config rows.
func (r *PostgresRepo) SetDeviceID(ctx context.Context, deviceID string) error {
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE %s SET device_id = $1 WHERE linea_id > 0`, r.tbl()), deviceID)
	return err
}

func (r *PostgresRepo) Close() error {
	return r.db.Close()
}
