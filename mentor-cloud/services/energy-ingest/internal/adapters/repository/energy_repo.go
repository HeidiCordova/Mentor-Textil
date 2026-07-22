package repository

import (
	"context"
	"energy-ingest/internal/domain"
	"energy-ingest/internal/ports"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EnergyRepo struct {
	db *pgxpool.Pool
}

func NewEnergyRepo(db *pgxpool.Pool) *EnergyRepo {
	return &EnergyRepo{db: db}
}

func (r *EnergyRepo) UpsertSnapshot(ctx context.Context, s *domain.EnergySnapshot) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO energy.snapshots
		(device_id, meter_id, hora, interval_s, head, data,
		 empresa_id, planta_id,
		 corriente_a, corriente_b, corriente_c, corriente_avg,
		 voltaje_a, voltaje_b, voltaje_c, voltaje_avg,
		 voltaje_ab, voltaje_bc, voltaje_ac,
		 potencia_activa, potencia_reactiva, potencia_aparente, factor_potencia,
		 frecuencia_hz,
		 energia_activa, energia_reactiva, energia_aparente,
		 thd_ia, thd_ib, thd_ic, thd_ua, thd_ub, thd_uc)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
		        $9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,
		        $20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33)
		ON CONFLICT (device_id, meter_id, hora) DO UPDATE SET
			interval_s        = EXCLUDED.interval_s,
			head              = EXCLUDED.head,
			data              = EXCLUDED.data,
			-- Si el snapshot fue insertado sin scope (device no registrado), se
			-- corrige al recibir el reintento una vez que el device esté registrado.
			empresa_id        = COALESCE(EXCLUDED.empresa_id, snapshots.empresa_id),
			planta_id         = COALESCE(EXCLUDED.planta_id, snapshots.planta_id),
			corriente_a       = EXCLUDED.corriente_a,
			corriente_b       = EXCLUDED.corriente_b,
			corriente_c       = EXCLUDED.corriente_c,
			corriente_avg     = EXCLUDED.corriente_avg,
			voltaje_a         = EXCLUDED.voltaje_a,
			voltaje_b         = EXCLUDED.voltaje_b,
			voltaje_c         = EXCLUDED.voltaje_c,
			voltaje_avg       = EXCLUDED.voltaje_avg,
			voltaje_ab        = EXCLUDED.voltaje_ab,
			voltaje_bc        = EXCLUDED.voltaje_bc,
			voltaje_ac        = EXCLUDED.voltaje_ac,
			potencia_activa   = EXCLUDED.potencia_activa,
			potencia_reactiva = EXCLUDED.potencia_reactiva,
			potencia_aparente = EXCLUDED.potencia_aparente,
			factor_potencia   = EXCLUDED.factor_potencia,
			frecuencia_hz     = EXCLUDED.frecuencia_hz,
			energia_activa    = EXCLUDED.energia_activa,
			energia_reactiva  = EXCLUDED.energia_reactiva,
			energia_aparente  = EXCLUDED.energia_aparente,
			thd_ia            = EXCLUDED.thd_ia,
			thd_ib            = EXCLUDED.thd_ib,
			thd_ic            = EXCLUDED.thd_ic,
			thd_ua            = EXCLUDED.thd_ua,
			thd_ub            = EXCLUDED.thd_ub,
			thd_uc            = EXCLUDED.thd_uc
		RETURNING id`,
		s.DeviceID, s.MeterID, s.Hora, s.IntervalS, s.Head, s.Data,
		s.EmpresaID, s.PlantaID,
		s.CorrienteA, s.CorrienteB, s.CorrienteC, s.CorrienteAVG,
		s.VoltajeA, s.VoltajeB, s.VoltajeC, s.VoltajeAVG,
		s.VoltajeAB, s.VoltajeBC, s.VoltajeAC,
		s.PotenciaActiva, s.PotenciaReactiva, s.PotenciaAparente, s.FactorPotencia,
		s.FrecuenciaHz,
		s.EnergiaActiva, s.EnergiaReactiva, s.EnergiaAparente,
		s.THDIA, s.THDIB, s.THDIC, s.THDUA, s.THDUB, s.THDUC,
	).Scan(&s.ID)
}

func (r *EnergyRepo) UpsertMeter(ctx context.Context, m *domain.Meter) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO energy.meters (device_id, meter_id, empresa_id, planta_id, ubicacion, activo)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (device_id, meter_id) DO UPDATE SET
			empresa_id = EXCLUDED.empresa_id,
			planta_id  = EXCLUDED.planta_id,
			ubicacion  = COALESCE(EXCLUDED.ubicacion, energy.meters.ubicacion),
			activo     = EXCLUDED.activo,
			updated_at = NOW()`,
		m.DeviceID, m.MeterID, m.EmpresaID, m.PlantaID, m.Ubicacion, m.Activo,
	)
	return err
}

func (r *EnergyRepo) LogSync(ctx context.Context, l *domain.DeviceSyncLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO energy.device_sync_log (device_id, batch_size, status, error_msg)
		VALUES ($1,$2,$3,$4)`,
		l.DeviceID, l.BatchSize, l.Status, l.ErrorMsg,
	)
	return err
}

func (r *EnergyRepo) GetSnapshots(ctx context.Context, f ports.SnapshotFilter) ([]domain.EnergySnapshot, int, error) {
	var conds []string
	var args []interface{}
	idx := 1

	if f.DeviceID != "" {
		conds = append(conds, fmt.Sprintf("device_id = $%d", idx))
		args = append(args, f.DeviceID)
		idx++
	}
	if f.MeterID != "" {
		conds = append(conds, fmt.Sprintf("meter_id = $%d", idx))
		args = append(args, f.MeterID)
		idx++
	}
	if f.EmpresaID != nil {
		conds = append(conds, fmt.Sprintf("empresa_id = $%d", idx))
		args = append(args, *f.EmpresaID)
		idx++
	}
	if f.PlantaID != nil {
		conds = append(conds, fmt.Sprintf("planta_id = $%d", idx))
		args = append(args, *f.PlantaID)
		idx++
	}
	if f.From != "" {
		conds = append(conds, fmt.Sprintf("hora >= $%d::timestamptz", idx))
		args = append(args, f.From)
		idx++
	}
	if f.To != "" {
		conds = append(conds, fmt.Sprintf("hora <= $%d::timestamptz", idx))
		args = append(args, f.To)
		idx++
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM energy.snapshots %s", where)
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if f.Limit <= 0 {
		f.Limit = 50
	}

	// CTE base: calcula el snapshot anterior por medidor usando LAG().
	// El PARTITION BY (device_id, meter_id) garantiza que el delta no cruza entre
	// medidores del mismo nombre en distintos dispositivos (distintas RPis).
	// El ORDER BY hora ASC en la ventana asegura que prev = el snapshot inmediatamente
	// anterior en el tiempo. El CASE descarta deltas negativos (reset de contador).
	dataQ := fmt.Sprintf(`
		WITH base AS (
			SELECT id, device_id, meter_id, hora, interval_s, head, data,
			       empresa_id, planta_id,
			       corriente_a, corriente_b, corriente_c, corriente_avg,
			       voltaje_a, voltaje_b, voltaje_c, voltaje_avg,
			       voltaje_ab, voltaje_bc, voltaje_ac,
			       potencia_activa, potencia_reactiva, potencia_aparente, factor_potencia,
			       frecuencia_hz,
			       energia_activa, energia_reactiva, energia_aparente,
			       thd_ia, thd_ib, thd_ic, thd_ua, thd_ub, thd_uc,
			       created_at,
			       LAG(energia_activa)   OVER (PARTITION BY device_id, meter_id ORDER BY hora ASC) AS prev_ea,
			       LAG(energia_reactiva) OVER (PARTITION BY device_id, meter_id ORDER BY hora ASC) AS prev_er,
			       LAG(energia_aparente) OVER (PARTITION BY device_id, meter_id ORDER BY hora ASC) AS prev_es
			  FROM energy.snapshots %s
		)
		SELECT id, device_id, meter_id, hora, interval_s, head, data,
		       empresa_id, planta_id,
		       corriente_a, corriente_b, corriente_c, corriente_avg,
		       voltaje_a, voltaje_b, voltaje_c, voltaje_avg,
		       voltaje_ab, voltaje_bc, voltaje_ac,
		       potencia_activa, potencia_reactiva, potencia_aparente, factor_potencia,
		       frecuencia_hz,
		       energia_activa, energia_reactiva, energia_aparente,
		       thd_ia, thd_ib, thd_ic, thd_ua, thd_ub, thd_uc,
		       created_at,
		       CASE WHEN prev_ea IS NOT NULL AND energia_activa - prev_ea >= 0
		            THEN energia_activa - prev_ea ELSE NULL END AS consumo_activa,
		       CASE WHEN prev_er IS NOT NULL AND energia_reactiva - prev_er >= 0
		            THEN energia_reactiva - prev_er ELSE NULL END AS consumo_reactiva,
		       CASE WHEN prev_es IS NOT NULL AND energia_aparente - prev_es >= 0
		            THEN energia_aparente - prev_es ELSE NULL END AS consumo_aparente
		  FROM base
		 ORDER BY hora DESC
		 LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	args = append(args, f.Limit, f.Offset)

	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []domain.EnergySnapshot
	for rows.Next() {
		var s domain.EnergySnapshot
		if err := rows.Scan(
			&s.ID, &s.DeviceID, &s.MeterID, &s.Hora, &s.IntervalS, &s.Head, &s.Data,
			&s.EmpresaID, &s.PlantaID,
			&s.CorrienteA, &s.CorrienteB, &s.CorrienteC, &s.CorrienteAVG,
			&s.VoltajeA, &s.VoltajeB, &s.VoltajeC, &s.VoltajeAVG,
			&s.VoltajeAB, &s.VoltajeBC, &s.VoltajeAC,
			&s.PotenciaActiva, &s.PotenciaReactiva, &s.PotenciaAparente, &s.FactorPotencia,
			&s.FrecuenciaHz,
			&s.EnergiaActiva, &s.EnergiaReactiva, &s.EnergiaAparente,
			&s.THDIA, &s.THDIB, &s.THDIC, &s.THDUA, &s.THDUB, &s.THDUC,
			&s.CreatedAt,
			&s.ConsumoActiva, &s.ConsumoReactiva, &s.ConsumoAparente,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, s)
	}

	return result, total, nil
}

func (r *EnergyRepo) GetMeters(ctx context.Context, empresaID *int) ([]domain.Meter, error) {
	q := `SELECT id, device_id, meter_id, empresa_id, planta_id, nombre, ubicacion, activo, created_at, updated_at
	        FROM energy.meters`
	var args []interface{}
	if empresaID != nil {
		q += " WHERE empresa_id = $1"
		args = append(args, *empresaID)
	}
	q += " ORDER BY device_id, meter_id"

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Meter
	for rows.Next() {
		var m domain.Meter
		if err := rows.Scan(
			&m.ID, &m.DeviceID, &m.MeterID, &m.EmpresaID, &m.PlantaID,
			&m.Nombre, &m.Ubicacion, &m.Activo, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *EnergyRepo) GetMeterByID(ctx context.Context, id int) (*domain.Meter, error) {
	var m domain.Meter
	err := r.db.QueryRow(ctx,
		`SELECT id, device_id, meter_id, empresa_id, planta_id, nombre, ubicacion, activo, created_at, updated_at
		   FROM energy.meters WHERE id = $1`, id,
	).Scan(&m.ID, &m.DeviceID, &m.MeterID, &m.EmpresaID, &m.PlantaID,
		&m.Nombre, &m.Ubicacion, &m.Activo, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *EnergyRepo) UpdateMeter(ctx context.Context, id int, m *domain.MeterUpdate) error {
	_, err := r.db.Exec(ctx, `
		UPDATE energy.meters
		SET nombre    = COALESCE($2, nombre),
		    ubicacion = COALESCE($3, ubicacion),
		    activo    = COALESCE($4, activo),
		    updated_at = NOW()
		WHERE id = $1`,
		id, m.Nombre, m.Ubicacion, m.Activo,
	)
	return err
}

func (r *EnergyRepo) DeleteMeter(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM energy.meters WHERE id = $1`, id)
	return err
}
