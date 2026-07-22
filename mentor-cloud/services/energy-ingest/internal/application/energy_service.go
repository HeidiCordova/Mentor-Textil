package application

import (
	"context"
	"energy-ingest/internal/adapters/repository"
	"energy-ingest/internal/domain"
	"energy-ingest/internal/ports"
	"fmt"
	"log"
	"time"
)

type EnergyService struct {
	repo       ports.EnergyRepository
	plantaRepo *repository.PlantaEnergyRepo // nil cuando multitenancy no configurado
	scopes     ports.ScopeResolver
}

func NewEnergyService(repo ports.EnergyRepository, plantaRepo *repository.PlantaEnergyRepo, scopes ports.ScopeResolver) *EnergyService {
	return &EnergyService{repo: repo, plantaRepo: plantaRepo, scopes: scopes}
}

func (s *EnergyService) ProcessBatch(ctx context.Context, deviceID string, records []domain.EnergyRecord) (int, error) {
	scope, scopeErr := s.scopes.ResolveByDevice(ctx, deviceID)
	if scopeErr != nil {
		// Sin scope, los snapshots se insertan con empresa/planta nulos y quedan
		// invisibles en el dashboard de empresa. Registrar el device en
		// gateway.device_registry para resolver este problema.
		log.Printf("[energy-ingest] WARN: scope no resuelto device=%s: %v — datos sin empresa/planta", deviceID, scopeErr)
	}
	go s.scopes.UpdateLastSeen(context.Background(), deviceID)

	// Determinar si podemos escribir en la BD de planta (Option B).
	canPlanta := s.plantaRepo != nil && scope != nil && scope.PlantaID != nil && scope.LineaID != nil

	inserted := 0
	for _, rec := range records {
		ts := time.Unix(rec.Timestamp, 0)
		fields := domain.ExtractFields(rec.Head, rec.Data)

		snap := &domain.EnergySnapshot{
			DeviceID:         deviceID,
			MeterID:          rec.MeterID,
			Hora:             ts,
			IntervalS:        rec.IntervalS,
			Head:             rec.Head,
			Data:             rec.Data,
			CorrienteA:       fields["corriente_a"],
			CorrienteB:       fields["corriente_b"],
			CorrienteC:       fields["corriente_c"],
			CorrienteAVG:     fields["corriente_avg"],
			VoltajeA:         fields["voltaje_a"],
			VoltajeB:         fields["voltaje_b"],
			VoltajeC:         fields["voltaje_c"],
			VoltajeAVG:       fields["voltaje_avg"],
			VoltajeAB:        fields["voltaje_ab"],
			VoltajeBC:        fields["voltaje_bc"],
			VoltajeAC:        fields["voltaje_ac"],
			PotenciaActiva:   fields["potencia_activa"],
			PotenciaReactiva: fields["potencia_reactiva"],
			PotenciaAparente: fields["potencia_aparente"],
			FactorPotencia:   fields["factor_potencia"],
			FrecuenciaHz:     fields["frecuencia_hz"],
			EnergiaActiva:    fields["energia_activa"],
			EnergiaReactiva:  fields["energia_reactiva"],
			EnergiaAparente:  fields["energia_aparente"],
			THDIA:            fields["thd_ia"],
			THDIB:            fields["thd_ib"],
			THDIC:            fields["thd_ic"],
			THDUA:            fields["thd_ua"],
			THDUB:            fields["thd_ub"],
			THDUC:            fields["thd_uc"],
		}

		if scope != nil {
			snap.PlantaID = scope.PlantaID
			snap.EmpresaID = scope.EmpresaID
		}

		// --- Escritura primaria: BD de planta (Option B) ---
		if canPlanta {
			if err := s.plantaRepo.UpsertSnapshot(ctx, snap, *scope.PlantaID, *scope.LineaID); err != nil {
				log.Printf("planta write snapshot device=%s meter=%s: %v", deviceID, rec.MeterID, err)
			}
		}

		// --- Escritura secundaria: schema global energy.* (backward compat dashboard) ---
		if err := s.repo.UpsertSnapshot(ctx, snap); err != nil {
			return inserted, fmt.Errorf("upsert snapshot device=%s meter=%s: %w", deviceID, rec.MeterID, err)
		}
		inserted++

		meter := &domain.Meter{
			DeviceID: deviceID,
			MeterID:  rec.MeterID,
			Activo:   true,
		}
		if rec.Ubicacion != "" {
			meter.Ubicacion = &rec.Ubicacion
		}
		if scope != nil {
			meter.PlantaID = scope.PlantaID
			meter.EmpresaID = scope.EmpresaID
		}

		if canPlanta {
			if err := s.plantaRepo.UpsertMeter(ctx, meter, *scope.PlantaID, *scope.LineaID); err != nil {
				log.Printf("planta write meter %s/%s: %v", deviceID, rec.MeterID, err)
			}
		}

		if err := s.repo.UpsertMeter(ctx, meter); err != nil {
			log.Printf("upsert meter %s/%s: %v", deviceID, rec.MeterID, err)
		}
	}

	syncLog := &domain.DeviceSyncLog{
		DeviceID:  deviceID,
		BatchSize: inserted,
		Status:    "ok",
	}
	if canPlanta {
		if err := s.plantaRepo.LogSync(ctx, deviceID, inserted, "ok", "", *scope.PlantaID, *scope.LineaID); err != nil {
			log.Printf("planta log sync %s: %v", deviceID, err)
		}
	}
	_ = s.repo.LogSync(ctx, syncLog)

	return inserted, nil
}

func (s *EnergyService) GetSnapshots(ctx context.Context, filter ports.SnapshotFilter) ([]domain.EnergySnapshot, int, error) {
	return s.repo.GetSnapshots(ctx, filter)
}

func (s *EnergyService) GetMeters(ctx context.Context, empresaID *int) ([]domain.Meter, error) {
	return s.repo.GetMeters(ctx, empresaID)
}

func (s *EnergyService) GetMeterByID(ctx context.Context, id int) (*domain.Meter, error) {
	return s.repo.GetMeterByID(ctx, id)
}

func (s *EnergyService) CreateMeter(ctx context.Context, m *domain.MeterCreate) (*domain.Meter, error) {
	meter := &domain.Meter{
		DeviceID:  m.DeviceID,
		MeterID:   m.MeterID,
		Activo:    true,
		Nombre:    &m.Nombre,
		Ubicacion: &m.Ubicacion,
		EmpresaID: m.EmpresaID,
		PlantaID:  m.PlantaID,
	}
	if err := s.repo.UpsertMeter(ctx, meter); err != nil {
		return nil, err
	}
	return meter, nil
}

func (s *EnergyService) UpdateMeter(ctx context.Context, id int, m *domain.MeterUpdate) error {
	return s.repo.UpdateMeter(ctx, id, m)
}

func (s *EnergyService) DeleteMeter(ctx context.Context, id int) error {
	return s.repo.DeleteMeter(ctx, id)
}
