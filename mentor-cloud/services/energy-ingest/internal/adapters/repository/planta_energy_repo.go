package repository

import (
	"context"
	"energy-ingest/internal/domain"
	"fmt"

	"mentor.local/shared/multitenancy"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PlantaEnergyRepo escribe snapshots de energía en la BD de planta
// (mentor_planta_X.linea_Y.snapshots) en vez del schema global energy.
type PlantaEnergyRepo struct {
	mgr *multitenancy.PlantaPoolManager
}

func NewPlantaEnergyRepo(mgr *multitenancy.PlantaPoolManager) *PlantaEnergyRepo {
	return &PlantaEnergyRepo{mgr: mgr}
}

func (r *PlantaEnergyRepo) pool(ctx context.Context, plantaID int) (*pgxpool.Pool, error) {
	return r.mgr.Get(ctx, plantaID)
}

func tbl(lineaID int, name string) string {
	return fmt.Sprintf("linea_%d.%s", lineaID, name)
}

func (r *PlantaEnergyRepo) UpsertSnapshot(ctx context.Context, s *domain.EnergySnapshot, plantaID, lineaID int) error {
	pool, err := r.pool(ctx, plantaID)
	if err != nil {
		return fmt.Errorf("pool planta %d: %w", plantaID, err)
	}
	table := tbl(lineaID, "snapshots")
	return pool.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO %s
		(device_id, meter_id, hora, interval_s, head, data,
		 corriente_a, corriente_b, corriente_c, corriente_avg,
		 voltaje_a, voltaje_b, voltaje_c, voltaje_avg,
		 voltaje_ab, voltaje_bc, voltaje_ac,
		 potencia_activa, potencia_reactiva, potencia_aparente, factor_potencia,
		 frecuencia_hz,
		 energia_activa, energia_reactiva, energia_aparente,
		 thd_ia, thd_ib, thd_ic, thd_ua, thd_ub, thd_uc)
		VALUES ($1,$2,$3,$4,$5,$6,
		        $7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,
		        $18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31)
		ON CONFLICT (device_id, meter_id, hora) DO UPDATE SET
			interval_s        = EXCLUDED.interval_s,
			head              = EXCLUDED.head,
			data              = EXCLUDED.data,
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
		RETURNING id`, table),
		s.DeviceID, s.MeterID, s.Hora, s.IntervalS, s.Head, s.Data,
		s.CorrienteA, s.CorrienteB, s.CorrienteC, s.CorrienteAVG,
		s.VoltajeA, s.VoltajeB, s.VoltajeC, s.VoltajeAVG,
		s.VoltajeAB, s.VoltajeBC, s.VoltajeAC,
		s.PotenciaActiva, s.PotenciaReactiva, s.PotenciaAparente, s.FactorPotencia,
		s.FrecuenciaHz,
		s.EnergiaActiva, s.EnergiaReactiva, s.EnergiaAparente,
		s.THDIA, s.THDIB, s.THDIC, s.THDUA, s.THDUB, s.THDUC,
	).Scan(&s.ID)
}

func (r *PlantaEnergyRepo) UpsertMeter(ctx context.Context, m *domain.Meter, plantaID, lineaID int) error {
	pool, err := r.pool(ctx, plantaID)
	if err != nil {
		return fmt.Errorf("pool planta %d: %w", plantaID, err)
	}
	table := tbl(lineaID, "meters")
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (device_id, meter_id, ubicacion, activo)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (device_id, meter_id) DO UPDATE SET
			ubicacion  = COALESCE(EXCLUDED.ubicacion, %s.ubicacion),
			activo     = EXCLUDED.activo,
			updated_at = NOW()`, table, table),
		m.DeviceID, m.MeterID, m.Ubicacion, m.Activo,
	)
	return err
}

func (r *PlantaEnergyRepo) LogSync(ctx context.Context, deviceID string, batchSize int, status, errMsg string, plantaID, lineaID int) error {
	pool, err := r.pool(ctx, plantaID)
	if err != nil {
		return err
	}
	table := tbl(lineaID, "device_sync_log")
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (device_id, batch_size, status, error_msg)
		VALUES ($1,$2,$3,$4)`, table),
		deviceID, batchSize, status, errMsg,
	)
	return err
}
