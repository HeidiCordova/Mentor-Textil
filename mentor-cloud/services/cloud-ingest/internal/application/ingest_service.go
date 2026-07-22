package application

import (
	"cloud-ingest/internal/domain"
	"cloud-ingest/internal/ports"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"mentor.local/shared/multitenancy"
)

type IngestService struct {
	events    ports.EventRepository
	notifier  ports.Notifier
	velocidad ports.VelocidadNominalRepo
}

func NewIngestService(events ports.EventRepository, notifier ports.Notifier, velocidad ports.VelocidadNominalRepo) *IngestService {
	return &IngestService{events: events, notifier: notifier, velocidad: velocidad}
}

func (s *IngestService) ProcessBatch(ctx context.Context, deviceID string, records []domain.OEERecord, scope *domain.DeviceScope) error {
	// Inyectar scope en el contexto para que PoolResolver resuelva el pool y schema
	// correctos de la planta (linea_X.velocidad_nominal) en vez del fallback legacy
	// (config.velocidad_nominal) al calcular la velocidad nominal de OEE.
	if scope.PlantaID != nil && scope.LineaID != nil {
		ctx = multitenancy.WithScope(ctx, multitenancy.Scope{
			PlantaID: *scope.PlantaID,
			LineaID:  *scope.LineaID,
		})
	}
	var inserted int
	for _, rec := range records {
		ts := time.UnixMilli(rec.Time)
		payload, _ := json.Marshal(rec)

		raw := &domain.RawEvent{
			DeviceID:      deviceID,
			EmpresaID:     scope.EmpresaID,
			PlantaID:      scope.PlantaID,
			LineaID:       scope.LineaID,
			EventType:     "oee",
			Payload:       payload,
			TimestampEdge: ts,
		}
		if err := s.events.InsertRawEvent(ctx, raw); err != nil {
			return fmt.Errorf("raw event insert device=%s: %w", deviceID, err)
		}
		inserted++

		snapshot := s.buildSnapshot(ctx, deviceID, rec, ts, scope.LineaID)
		snapshot.EmpresaID = scope.EmpresaID
		snapshot.PlantaID = scope.PlantaID
		snapshot.LineaID = scope.LineaID
		if err := s.events.InsertOEESnapshot(ctx, snapshot); err != nil {
			return fmt.Errorf("oee snapshot insert device=%s: %w", deviceID, err)
		}

		if len(rec.Head) > 0 {
			if err := s.events.UpdateVariablesFromHeadData(ctx, deviceID, rec.Head, rec.Data); err != nil {
				log.Printf("ingest: update variables from snapshot: %v", err)
			} else if s.notifier != nil && scope.EmpresaID != nil {
				go func(eid int) {
					vars, err := s.events.GetVariablesForEmpresa(context.Background(), eid)
					if err != nil {
						log.Printf("ingest: get variables for broadcast: %v", err)
						return
					}
					s.notifier.BroadcastVariables(context.Background(), int64(eid), vars)
				}(*scope.EmpresaID)
			}
		}
	}

	syncLog := &domain.DeviceSyncLog{
		DeviceID:  deviceID,
		BatchSize: inserted,
		Status:    "ok",
	}
	_ = s.events.LogSync(ctx, syncLog)

	return nil
}

func (s *IngestService) buildSnapshot(ctx context.Context, deviceID string, rec domain.OEERecord, ts time.Time, lineaID *int) *domain.OEESnapshot {
	snap := &domain.OEESnapshot{
		DeviceID:  deviceID, // usar siempre el device_id autoritativo del gateway, no el del payload
		Turno:     rec.Turno,
		Fecha:     ts.Format("2006-01-02"),
		Hora:      ts,
		Code:      rec.Code,
		IntervalS: rec.IntervalS,
		Head:      rec.Head,
		Data:      rec.Data,
	}

	headIndex := make(map[string]int, len(rec.Head))
	for i, h := range rec.Head {
		headIndex[h] = i
	}
	val := func(key string) float64 {
		if idx, ok := headIndex[key]; ok && idx < len(rec.Data) {
			f, err := strconv.ParseFloat(rec.Data[idx], 64)
			if err != nil {
				return 0
			}
			return f
		}
		return 0
	}

	raw := domain.OEERaw{
		TDisponibleTotal:          val("T_DISPONIBLE"),
		TMicroparada:              val("T_MICROPARADA"),
		TParadaNoAsignada:         val("T_PARADA_NO_ASIGNADA"),
		TParadaProgramada:         val("T_PARADA_PROGRAMADA"),
		TParadaNoProgramada:       val("T_PARADA_NO_PROGRAMADA"),
		TRefrigerio:               val("T_REFRIGERIO"),
		TCapacitacionObligatoria:  val("T_CAPACITACION_OBLIGATORIA"),
		TMantenimientoPlanificado: val("T_MANTENIMIENTO_PLANIFICADO"),
		Conteo1:                   val("CONTEO_1"),
		Merma:                     val("MERMA"),
	}

	var velocidadUS float64
	if lineaID != nil && s.velocidad != nil {
		v, err := s.velocidad.GetActiveVelocidad(ctx, *lineaID, ts)
		if err != nil {
			log.Printf("ingest: velocidad lookup linea=%d: %v", *lineaID, err)
		} else {
			velocidadUS = v
		}
	}

	result := domain.CalculateOEE(raw, velocidadUS)
	snap.Disponibilidad = result.Disponibilidad
	snap.Rendimiento = result.Rendimiento
	snap.Calidad = result.Calidad
	snap.OEE = result.OEE
	snap.Produccion = result.Produccion
	snap.EnergiaKwh = val("energia_kwh")

	return snap
}

func (s *IngestService) GetRawEvents(ctx context.Context, filter ports.RawEventsFilter) ([]domain.RawEvent, int, error) {
	return s.events.GetRawEvents(ctx, filter)
}

func (s *IngestService) ProcessStopsBatch(ctx context.Context, deviceID string, stops []domain.StopRecord, scope *domain.DeviceScope) (int, error) {
	inserted := 0
	for _, stop := range stops {
		// usar siempre el device_id autoritativo del gateway (no el del payload edge)
		stop.DeviceID = deviceID
		if err := s.events.UpsertParada(ctx, &stop, scope); err != nil {
			return inserted, fmt.Errorf("upsert parada %s: %w", stop.StopID, err)
		}
		inserted++
	}

	syncLog := &domain.DeviceSyncLog{
		DeviceID:  deviceID,
		BatchSize: inserted,
		Status:    "ok",
		Mensaje:   "stops_sync",
	}
	_ = s.events.LogSync(ctx, syncLog)

	return inserted, nil
}

func (s *IngestService) ProcessProductionRunsBatch(ctx context.Context, deviceID string, runs []domain.ProductionRunRecord, scope *domain.DeviceScope) (int, error) {
	upserted := 0
	for _, run := range runs {
		// usar siempre el device_id autoritativo del gateway (no el del payload edge)
		run.DeviceID = deviceID
		if err := s.events.UpsertProductionRun(ctx, &run, scope); err != nil {
			return upserted, fmt.Errorf("upsert production_run %s: %w", run.RunID, err)
		}
		upserted++
	}

	syncLog := &domain.DeviceSyncLog{
		DeviceID:  deviceID,
		BatchSize: upserted,
		Status:    "ok",
		Mensaje:   "production_runs_sync",
	}
	_ = s.events.LogSync(ctx, syncLog)

	return upserted, nil
}

// GetPendingCommands retorna los comandos pendientes para un dispositivo edge.
func (s *IngestService) GetPendingCommands(ctx context.Context, deviceID string) ([]domain.PendingCommand, error) {
	return s.events.GetPendingCommands(ctx, deviceID)
}

// AckPendingCommands marca los comandos dados como aplicados.
func (s *IngestService) AckPendingCommands(ctx context.Context, deviceID string, ids []int64) error {
	return s.events.AckPendingCommands(ctx, deviceID, ids)
}

// EnqueuePendingCommand agrega un comando cloud→edge a la cola confiable.
func (s *IngestService) EnqueuePendingCommand(ctx context.Context, deviceID string, command string, payload []byte) error {
	return s.events.InsertPendingCommand(ctx, deviceID, command, payload)
}
