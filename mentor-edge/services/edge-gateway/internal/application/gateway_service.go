package application

import (
	"context"
	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
	"fmt"
	"log"
	"time"
)

type GatewayService struct {
	stops          ports.StopRepository
	productionRuns ports.ProductionRunRepository
	catalogSync    ports.CatalogSyncRepository
	config         ports.ConfigClient
	buffer         ports.BufferClient
	detector       ports.DetectorClient
	enviador       ports.EnviadorClient
	audit          ports.AuditRepository
	broker         ports.SSEBroker
	deviceID       string
	startedAt      time.Time
}

func NewGatewayService(
	stops ports.StopRepository,
	productionRuns ports.ProductionRunRepository,
	catalogSync ports.CatalogSyncRepository,
	config ports.ConfigClient,
	buffer ports.BufferClient,
	detector ports.DetectorClient,
	enviador ports.EnviadorClient,
	audit ports.AuditRepository,
	broker ports.SSEBroker,
	deviceID string,
) *GatewayService {
	return &GatewayService{
		stops:          stops,
		productionRuns: productionRuns,
		catalogSync:    catalogSync,
		config:         config,
		buffer:         buffer,
		detector:       detector,
		enviador:       enviador,
		audit:          audit,
		broker:         broker,
		deviceID:       deviceID,
		startedAt:      time.Now(),
	}
}

func (s *GatewayService) AggregatedHealth(ctx context.Context) (*domain.AggregatedHealth, error) {
	health := &domain.AggregatedHealth{
		Service:  "edge-gateway",
		Status:   "ok",
		DeviceID: s.deviceID,
		Uptime:   int64(time.Since(s.startedAt).Seconds()),
		Deps:     make(map[string]string),
	}

	type depResult struct {
		name   string
		status string
	}

	ch := make(chan depResult, 4)

	go func() {
		st, err := s.buffer.Health(ctx)
		if err != nil {
			ch <- depResult{"resiliencia", "error"}
			return
		}
		ch <- depResult{"resiliencia", st}
	}()

	go func() {
		st, err := s.detector.Health(ctx)
		if err != nil {
			ch <- depResult{"detector", "error"}
			return
		}
		ch <- depResult{"detector", st}
	}()

	go func() {
		st, err := s.enviador.Health(ctx)
		if err != nil {
			ch <- depResult{"enviador", "error"}
			return
		}
		ch <- depResult{"enviador", st}
	}()

	go func() {
		_, err := s.config.GetConfigVersion(ctx)
		if err != nil {
			ch <- depResult{"config", "error"}
			return
		}
		ch <- depResult{"config", "ok"}
	}()

	for i := 0; i < 4; i++ {
		r := <-ch
		health.Deps[r.name] = r.status
		if r.status != "ok" {
			health.Status = "degraded"
		}
	}

	return health, nil
}

func (s *GatewayService) GetConfig(ctx context.Context) (map[string]interface{}, error) {
	cfg, err := s.config.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *GatewayService) UpdateConfig(ctx context.Context, patch map[string]interface{}, actor string) (map[string]interface{}, error) {
	result, err := s.config.UpdateConfig(ctx, patch)
	if err != nil {
		return nil, err
	}

	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID: s.deviceID,
		Actor:    actor,
		Action:   "UPDATE_CONFIG",
		Resource: "line_config",
		Payload:  patch,
		Result:   "OK",
	})

	return result, nil
}

func (s *GatewayService) StartCalibration(ctx context.Context, actor string) error {
	err := s.config.StartCalibration(ctx)
	if err != nil {
		return err
	}

	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID: s.deviceID,
		Actor:    actor,
		Action:   "START_CALIBRATION",
		Resource: "calibration",
		Result:   "OK",
	})
	return nil
}

func (s *GatewayService) CalibrationStatus(ctx context.Context) (map[string]interface{}, error) {
	return s.detector.CalibrationStatus(ctx)
}

func (s *GatewayService) BufferSummary(ctx context.Context) (*domain.BufferSummary, error) {
	return s.buffer.GetSummary(ctx)
}

func (s *GatewayService) RecentEvents(ctx context.Context, limit int, since *time.Time) ([]domain.Event, error) {
	if limit <= 0 || limit > 2000 {
		limit = 50
	}
	return s.buffer.GetRecentEvents(ctx, limit, since)
}

func (s *GatewayService) PendingEvents(ctx context.Context, limit int) ([]domain.Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	return s.buffer.GetPendingEvents(ctx, limit)
}

func (s *GatewayService) CreateStop(ctx context.Context, req domain.CreateStopRequest, actor string) (*domain.Stop, error) {
	if req.DeviceID == "" {
		req.DeviceID = s.deviceID
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	stop, err := s.stops.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID:   s.deviceID,
		Actor:      actor,
		Action:     "CREATE_STOP",
		Resource:   "stop",
		ResourceID: &stop.StopID,
		Result:     "OK",
	})

	s.broker.Publish("stop_created", map[string]any{"stop_id": stop.StopID, "device_id": stop.DeviceID})

	return stop, nil
}

func (s *GatewayService) GetStop(ctx context.Context, stopID string) (*domain.Stop, error) {
	return s.stops.GetByID(ctx, stopID)
}

func (s *GatewayService) JustifyStop(ctx context.Context, req domain.JustifyStopRequest, actor string) (*domain.Stop, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	stop, err := s.stops.Justify(ctx, req)
	if err != nil {
		return nil, err
	}

	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID:   s.deviceID,
		Actor:      actor,
		Action:     "JUSTIFY_STOP",
		Resource:   "stop",
		ResourceID: &stop.StopID,
		Payload:    map[string]interface{}{"reason": req.Reason, "category": req.Category},
		Result:     "OK",
	})

	s.broker.Publish("stop.changed", map[string]any{"stop_id": stop.StopID, "device_id": stop.DeviceID})

	return stop, nil
}

func (s *GatewayService) ListStops(ctx context.Context, filter domain.StopFilter) ([]domain.Stop, error) {
	// device_id is optional on the edge — caller decides whether to scope
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	return s.stops.List(ctx, filter)
}

func (s *GatewayService) CloseStop(ctx context.Context, stopID string, actor string) (*domain.Stop, error) {
	stop, err := s.stops.CloseStop(ctx, stopID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID:   s.deviceID,
		Actor:      actor,
		Action:     "CLOSE_STOP",
		Resource:   "stop",
		ResourceID: &stop.StopID,
		Result:     "OK",
	})
	return stop, nil
}

func (s *GatewayService) DeleteStop(ctx context.Context, stopID string, actor string) error {
	if err := s.stops.Delete(ctx, stopID); err != nil {
		return err
	}
	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID:   s.deviceID,
		Actor:      actor,
		Action:     "DELETE_STOP",
		Resource:   "stop",
		ResourceID: &stopID,
		Result:     "OK",
	})
	s.broker.Publish("stop.changed", map[string]any{"stop_id": stopID, "device_id": s.deviceID, "deleted": true})
	return nil
}

func (s *GatewayService) ProductsFromConfig(cfg map[string]interface{}) []domain.ProductEntry {
	raw, ok := cfg["products"]
	if !ok {
		return []domain.ProductEntry{}
	}
	list, ok := raw.([]interface{})
	if !ok {
		return []domain.ProductEntry{}
	}
	out := make([]domain.ProductEntry, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		entry := domain.ProductEntry{}
		if v, ok := m["sku"].(string); ok {
			entry.SKU = v
		}
		if v, ok := m["description"].(string); ok {
			entry.Description = v
		}
		if v, ok := m["active"].(bool); ok {
			entry.Active = v
		} else {
			entry.Active = true
		}
		out = append(out, entry)
	}
	return out
}

func (s *GatewayService) StopsSummary(ctx context.Context, hours int) (*domain.StopSummary, error) {
	if hours <= 0 {
		hours = 24
	}
	return s.stops.GetSummary(ctx, s.deviceID, hours)
}

// StopDurationByType returns OEE time fields in **seconds** for the last
// intervalS seconds. The map includes:
//
//	T_DISPONIBLE                 → seconds of the interval within active shifts (turnos)
//	T_PARADA_PROGRAMADA          → justified stops of type PROGRAMADA
//	T_PARADA_NO_PROGRAMADA       → justified stops of type NO_PROGRAMADA
//	T_REFRIGERIO                 → justified stops of type REFRIGERIO
//	T_CAPACITACION_OBLIGATORIA   → justified stops of type CAPACITACION
//	T_MANTENIMIENTO_PLANIFICADO  → justified stops of type MANTENIMIENTO
//	T_MICROPARADA                → justified stops of type MICROPARADA
func (s *GatewayService) StopDurationByType(ctx context.Context, intervalS int) (map[string]int64, error) {
	if intervalS <= 0 {
		intervalS = 60
	}
	raw, err := s.stops.GetDurationByType(ctx, s.deviceID, intervalS)
	if err != nil {
		return nil, err
	}
	// Map DB stop_type codes → OEE variable claves; convert duration_ms → segundos
	mapping := map[string]string{
		"PROGRAMADA":    "T_PARADA_PROGRAMADA",
		"NO_PROGRAMADA": "T_PARADA_NO_PROGRAMADA",
		"REFRIGERIO":    "T_REFRIGERIO",
		"CAPACITACION":  "T_CAPACITACION_OBLIGATORIA",
		"MANTENIMIENTO": "T_MANTENIMIENTO_PLANIFICADO",
		"MICROPARADA":   "T_MICROPARADA",
	}
	out := make(map[string]int64, len(mapping)+1)
	for dbType, clave := range mapping {
		out[clave] = raw[dbType] / 1000 // duration_ms → seconds
	}
	// T_DISPONIBLE: seconds of the interval that fall within active shift windows
	out["T_DISPONIBLE"] = s.shiftAvailableSeconds(ctx, time.Now(), intervalS)
	return out, nil
}

// shiftAvailableSeconds returns how many seconds of the last intervalS seconds
// fall within active shift windows (turnos), minus any seconds covered by
// "sin programación" production runs (sku IS NULL) in the same window.
// Falls back to intervalS when no shifts are configured.
func (s *GatewayService) shiftAvailableSeconds(ctx context.Context, now time.Time, intervalS int) int64 {
	windowStart := now.Add(-time.Duration(intervalS) * time.Second)

	turnos, err := s.catalogSync.ListTurnos(ctx)
	var shiftSecs int64
	if err != nil || len(turnos) == 0 {
		// No shifts configured: treat the full interval as shift time
		shiftSecs = int64(intervalS)
	} else {
		loc := now.Location()
		// Evaluate against both calendar days that the window spans (handles midnight crossing)
		days := shiftDays(windowStart, now, loc)
		for _, turno := range turnos {
			if !turno.Activo {
				continue
			}
			for _, day := range days {
				shStart := parseHHMMSS(turno.HoraInicio, day, loc)
				shEnd := parseHHMMSS(turno.HoraFin, day, loc)
				// HoraFin "00:00:00" means the shift ends at midnight of the next day
				if !shEnd.After(shStart) {
					shEnd = shEnd.Add(24 * time.Hour)
				}
				// Intersection of shift [shStart, shEnd) with window [windowStart, now)
				oStart := laterOf(shStart, windowStart)
				oEnd := earlierOf(shEnd, now)
				if oEnd.After(oStart) {
					shiftSecs += int64(oEnd.Sub(oStart).Seconds())
				}
			}
		}
		// Cap at intervalS — safeguard against overlapping shifts being double-counted
		if shiftSecs > int64(intervalS) {
			shiftSecs = int64(intervalS)
		}
	}

	// Subtract seconds covered by "sin programación" runs (sku IS NULL).
	// Example: 8h shift (28 800 s), 2h sin programación → T_DISPONIBLE = 21 600 s
	sinProgSecs, err := s.productionRuns.SinProgramacionSeconds(ctx, s.deviceID, windowStart, now)
	if err != nil {
		log.Printf("[gateway] shiftAvailableSeconds: sin_programacion query error: %v", err)
		sinProgSecs = 0
	}

	available := shiftSecs - sinProgSecs
	if available < 0 {
		available = 0
	}
	return available
}

// CurrentTurnoName returns the nombre of the active turno at the current time,
// or an empty string if no turno is active (or none are configured).
func (s *GatewayService) CurrentTurnoName(ctx context.Context) string {
	turnos, err := s.catalogSync.ListTurnos(ctx)
	if err != nil || len(turnos) == 0 {
		return ""
	}
	now := time.Now()
	loc := now.Location()
	for _, t := range turnos {
		if !t.Activo {
			continue
		}
		// Build shift window anchored to today; handle overnight shifts.
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		shStart := parseHHMMSS(t.HoraInicio, day, loc)
		shEnd := parseHHMMSS(t.HoraFin, day, loc)
		if !shEnd.After(shStart) {
			shEnd = shEnd.Add(24 * time.Hour)
		}
		if !now.Before(shStart) && now.Before(shEnd) {
			return t.Nombre
		}
		// Also check yesterday's window (in case of overnight shift that started yesterday).
		prevDay := day.Add(-24 * time.Hour)
		shStart2 := parseHHMMSS(t.HoraInicio, prevDay, loc)
		shEnd2 := parseHHMMSS(t.HoraFin, prevDay, loc)
		if !shEnd2.After(shStart2) {
			shEnd2 = shEnd2.Add(24 * time.Hour)
		}
		if !now.Before(shStart2) && now.Before(shEnd2) {
			return t.Nombre
		}
	}
	return ""
}

// parseHHMMSS parses a time string "HH:MM:SS" anchored to the given base day.
func parseHHMMSS(s string, day time.Time, loc *time.Location) time.Time {
	var h, m, sec int
	fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec)
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, sec, 0, loc)
}

// shiftDays returns the unique calendar days (00:00 midnight) spanned by [start, end).
func shiftDays(start, end time.Time, loc *time.Location) []time.Time {
	d0 := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	d1 := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
	if d0.Equal(d1) {
		return []time.Time{d0}
	}
	return []time.Time{d0, d1}
}

func laterOf(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func earlierOf(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (s *GatewayService) RunMaintenance(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.doMaintenance()
		}
	}
}

func (s *GatewayService) doMaintenance() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if n, err := s.stops.CloseStaleStops(ctx, 12*time.Hour); err != nil {
		log.Printf("[gateway] maintenance: close stale stops: %v", err)
	} else if n > 0 {
		log.Printf("[gateway] maintenance: closed %d stale stops", n)
	}

	if n, err := s.audit.PurgeOlderThan(ctx, 180); err != nil {
		log.Printf("[gateway] maintenance: purge audit: %v", err)
	} else if n > 0 {
		log.Printf("[gateway] maintenance: purged %d audit entries", n)
	}
}

func (s *GatewayService) UpsertProductionRun(ctx context.Context, req domain.UpsertProductionRunRequest, actor string) ([]domain.ProductionRun, error) {
	if req.DeviceID == "" {
		req.DeviceID = s.deviceID
	}
	runs, err := s.productionRuns.Upsert(ctx, req)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID: s.deviceID,
		Actor:    actor,
		Action:   "UPSERT_PRODUCTION_RUN",
		Resource: "production_run",
		Result:   "OK",
	})
	s.broker.Publish("production_runs_updated", map[string]any{"device_id": req.DeviceID})
	return runs, nil
}

func (s *GatewayService) ListProductionRuns(ctx context.Context, filter domain.ProductionRunFilter) ([]domain.ProductionRun, error) {
	if filter.DeviceID == "" {
		filter.DeviceID = s.deviceID
	}
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 200
	}
	return s.productionRuns.List(ctx, filter)
}

func (s *GatewayService) DeleteProductionRun(ctx context.Context, runID string, actor string) ([]domain.ProductionRun, error) {
	runs, err := s.productionRuns.Delete(ctx, runID)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID:   s.deviceID,
		Actor:      actor,
		Action:     "DELETE_PRODUCTION_RUN",
		Resource:   "production_run",
		ResourceID: &runID,
		Result:     "OK",
	})
	s.broker.Publish("production_runs_updated", map[string]any{"device_id": s.deviceID})
	return runs, nil
}
