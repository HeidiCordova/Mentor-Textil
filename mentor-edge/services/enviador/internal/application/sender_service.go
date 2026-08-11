package application

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"enviador/internal/domain"
	"enviador/internal/ports"
)

type CloudReporter interface {
	MarkAlive()
	MarkCloudOK()
	MarkError()
	IncrementSynced(n int)
}

type SenderService struct {
	storage         ports.EventStorage
	cloudClient     ports.CloudClient
	retryPolicy     *domain.RetryPolicy
	syncPolicy      *domain.SyncPolicy
	cloudReporter   CloudReporter
	lineaID         int
	lastConfigCheck time.Time
	lastSyncedMode  string
	currentCloudURL string
	currentAPIKey   string
}

func NewSenderService(
	storage ports.EventStorage,
	cloudClient ports.CloudClient,
	retryPolicy *domain.RetryPolicy,
	syncPolicy *domain.SyncPolicy,
	cloudReporter CloudReporter,
	lineaID int,
) *SenderService {
	return &SenderService{
		storage:       storage,
		cloudClient:   cloudClient,
		retryPolicy:   retryPolicy,
		syncPolicy:    syncPolicy,
		cloudReporter: cloudReporter,
		lineaID:       lineaID,
	}
}

func (s *SenderService) Run(ctx context.Context) {
	log.Printf("[enviador] linea_%d starting with sync interval: %ds", s.lineaID, s.syncPolicy.GetPollInterval())

	// Goroutine dedicado a sincronizar el mode cada 30s para reaccionar rápido a cambios.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		s.syncMode(ctx) // intento inicial inmediato
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.syncMode(ctx)
			}
		}
	}()

	// Goroutine dedicado a aplicar pending commands cada 3s (tiempo real).
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		s.fetchAndApplyPendingCommands(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.fetchAndApplyPendingCommands(ctx)
			}
		}
	}()

	// Goroutine dedicado a sincronizar stops (justificaciones) cada 3s — NO espera los 300s de OEE.
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		s.processStops(ctx)          // intento inicial inmediato
		s.processProductionRuns(ctx) // ídem para production runs
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.processStops(ctx)
				s.processProductionRuns(ctx)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		// Drain al arrancar: vacía todo el backlog acumulado durante la desconexión
		// antes de esperar el primer tick.
		for s.processEnergy(ctx) > 0 {}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Drenar la cola completa en cada ciclo; sin esto un backlog
				// grande tardaría N × 60 s en vaciarse (1 lote por tick).
				for s.processEnergy(ctx) > 0 {}
			}
		}
	}()

	for {
		if s.cloudReporter != nil {
			s.cloudReporter.MarkAlive()
		}
		s.refreshSyncInterval(ctx)
		s.processBatch(ctx)

		interval := time.Duration(s.syncPolicy.GetPollInterval()) * time.Second
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func (s *SenderService) refreshSyncInterval(ctx context.Context) {
	if time.Since(s.lastConfigCheck) < 30*time.Second {
		return
	}
	s.lastConfigCheck = time.Now()

	interval, err := s.storage.GetCloudSyncInterval(ctx, s.lineaID)
	if err == nil && interval > 0 && interval != s.syncPolicy.GetPollInterval() {
		log.Printf("[enviador] linea_%d sync interval updated: %ds", s.lineaID, interval)
		s.syncPolicy.SetPollInterval(interval)
	}

	cloudURL, apiKey, err := s.storage.GetCloudCredentials(ctx, s.lineaID)
	if err != nil {
		return
	}
	changed := false
	if cloudURL != "" && cloudURL != s.currentCloudURL {
		s.currentCloudURL = cloudURL
		changed = true
	}
	if apiKey != "" && apiKey != s.currentAPIKey {
		s.currentAPIKey = apiKey
		changed = true
	}
	if changed {
		log.Printf("[enviador] linea_%d cloud credentials updated", s.lineaID)
		s.cloudClient.UpdateCredentials(s.currentCloudURL, s.currentAPIKey)
	}
}

func (s *SenderService) processBatch(ctx context.Context) {
	events, err := s.storage.GetUnsyncedEvents(ctx, s.syncPolicy.GetBatchSize())
	if err != nil {
		log.Printf("Failed to get unsynced events: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	var oeeRecords []ports.OEERecord
	var oeeIDs []string
	var deadIDs []string

	for _, event := range events {
		if s.retryPolicy.IsEventExhausted(event.RetryCount) {
			log.Printf("OEE event %s exceeded %d retries, marking dead", event.EventID, event.RetryCount)
			deadIDs = append(deadIDs, event.EventID)
			continue
		}

		rec, err := s.parseOEEEvent(event)
		if err != nil {
			log.Printf("Failed to parse OEE event %s: %v", event.EventID, err)
			continue
		}
		oeeRecords = append(oeeRecords, rec)
		oeeIDs = append(oeeIDs, event.EventID)
	}

	if len(deadIDs) > 0 {
		if err := s.storage.MarkDead(ctx, deadIDs); err != nil {
			log.Printf("[enviador] warn: failed to mark %d events as dead: %v", len(deadIDs), err)
		}
	}

	if len(oeeRecords) > 0 {
		if err := s.sendOEEWithRetry(ctx, oeeRecords, oeeIDs); err != nil {
			log.Printf("OEE send failed after retries: %v", err)
		}
	}
}

func (s *SenderService) parseOEEEvent(event ports.Event) (ports.OEERecord, error) {
	// Usamos json.RawMessage para data porque puede contener enteros Y strings
	// (ej. MARCA="DASD", TAMANIO="324") mezclados con valores numéricos.
	var payload struct {
		Code      string            `json:"code"`
		DeviceID  string            `json:"device_id"`
		IntervalS int               `json:"interval_s"`
		Turno     string            `json:"turno"`
		V         int               `json:"v"`
		Head      []string          `json:"head"`
		Data      []json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return ports.OEERecord{}, err
	}

	// Convertir cada elemento a string: números → "1800", strings → "DASD".
	data := make([]string, len(payload.Data))
	for i, raw := range payload.Data {
		// Intentar como número primero
		var n json.Number
		if err := json.Unmarshal(raw, &n); err == nil {
			data[i] = n.String()
		} else {
			// Es un string JSON: deserializar para quitar comillas
			var s string
			if err := json.Unmarshal(raw, &s); err == nil {
				data[i] = s
			} else {
				data[i] = string(raw)
			}
		}
	}

	deviceID := payload.DeviceID
	if deviceID == "" {
		deviceID = event.DeviceID
	}

	return ports.OEERecord{
		Code:      payload.Code,
		Time:      event.Timestamp.UnixMilli(),
		DeviceID:  deviceID,
		IntervalS: payload.IntervalS,
		Turno:     payload.Turno,
		V:         payload.V,
		Head:      payload.Head,
		Data:      data,
	}, nil
}

func (s *SenderService) sendOEEWithRetry(ctx context.Context, records []ports.OEERecord, eventIDs []string) error {
	var lastErr error

	for attempt := 0; s.retryPolicy.ShouldRetryInline(attempt); attempt++ {
		if attempt > 0 {
			delay := s.retryPolicy.GetDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := s.cloudClient.SendOEE(ctx, records); err != nil {
			lastErr = err
			log.Printf("OEE send attempt %d failed: %v", attempt+1, err)
			continue
		}

		if err := s.storage.MarkSynced(ctx, eventIDs); err != nil {
			return err
		}

		if s.cloudReporter != nil {
			s.cloudReporter.MarkCloudOK()
			s.cloudReporter.IncrementSynced(len(eventIDs))
		}

		log.Printf("Synced %d OEE records to cloud", len(records))
		return nil
	}

	for _, eventID := range eventIDs {
		if err := s.storage.IncrementRetry(ctx, eventID); err != nil {
			log.Printf("[enviador] warn: failed to increment retry for event %s: %v", eventID, err)
		}
	}

	if s.cloudReporter != nil {
		s.cloudReporter.MarkError()
	}

	return lastErr
}

func (s *SenderService) processProductionRuns(ctx context.Context) {
	runs, err := s.storage.GetUnsyncedProductionRuns(ctx, s.syncPolicy.GetBatchSize())
	if err != nil {
		log.Printf("Failed to get unsynced production runs: %v", err)
		return
	}

	if len(runs) == 0 {
		return
	}

	if err := s.sendProductionRunsWithRetry(ctx, runs); err != nil {
		log.Printf("Production runs sync failed after retries: %v", err)
	}
}

func (s *SenderService) sendProductionRunsWithRetry(ctx context.Context, runs []ports.ProductionRunRecord) error {
	var lastErr error

	for attempt := 0; s.retryPolicy.ShouldRetryInline(attempt); attempt++ {
		if attempt > 0 {
			delay := s.retryPolicy.GetDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := s.cloudClient.SendProductionRuns(ctx, runs); err != nil {
			lastErr = err
			log.Printf("Production runs send attempt %d failed: %v", attempt+1, err)
			continue
		}

		ids := make([]string, len(runs))
		for i, r := range runs {
			ids[i] = r.RunID
		}
		if err := s.storage.MarkProductionRunsSynced(ctx, ids); err != nil {
			return err
		}

		if s.cloudReporter != nil {
			s.cloudReporter.MarkCloudOK()
			s.cloudReporter.IncrementSynced(len(runs))
		}

		log.Printf("Synced %d production runs to cloud", len(runs))
		return nil
	}

	if s.cloudReporter != nil {
		s.cloudReporter.MarkError()
	}

	return lastErr
}

func (s *SenderService) processStops(ctx context.Context) {
	stops, err := s.storage.GetUnsyncedStops(ctx, s.syncPolicy.GetBatchSize())
	if err != nil {
		log.Printf("Failed to get unsynced stops: %v", err)
		return
	}

	if len(stops) == 0 {
		return
	}

	// Convert ports.StopRecord to the same structure expected by cloud client
	if err := s.sendStopsWithRetry(ctx, stops); err != nil {
		log.Printf("Stops sync failed after retries: %v", err)
		return
	}
}

func (s *SenderService) sendStopsWithRetry(ctx context.Context, stops []ports.StopRecord) error {
	var lastErr error

	for attempt := 0; s.retryPolicy.ShouldRetryInline(attempt); attempt++ {
		if attempt > 0 {
			delay := s.retryPolicy.GetDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := s.cloudClient.SendStops(ctx, stops); err != nil {
			lastErr = err
			log.Printf("Stops send attempt %d failed: %v", attempt+1, err)
			continue
		}

		// Mark all sent stops as synced
		ids := make([]string, len(stops))
		for i, st := range stops {
			ids[i] = st.StopID
		}
		if err := s.storage.MarkStopsSynced(ctx, ids); err != nil {
			return err
		}

		if s.cloudReporter != nil {
			s.cloudReporter.MarkCloudOK()
			s.cloudReporter.IncrementSynced(len(stops))
		}

		log.Printf("Synced %d stops to cloud", len(stops))
		return nil
	}

	if s.cloudReporter != nil {
		s.cloudReporter.MarkError()
	}

	return lastErr
}

// fetchAndApplyPendingCommands descarga comandos pendientes del cloud y los aplica en el edge DB.
// Esto garantiza que justificaciones realizadas en cloud lleguen al edge aunque
// la conexión SSE estuviera caída en el momento de la justificación.
func (s *SenderService) fetchAndApplyPendingCommands(ctx context.Context) {
	cmds, err := s.cloudClient.GetPendingCommands(ctx)
	if err != nil {
		log.Printf("[enviador] pending commands fetch failed: %v", err)
		return
	}
	if len(cmds) == 0 {
		return
	}

	var appliedIDs []int64
	for _, cmd := range cmds {
		if err := s.storage.ApplyPendingCommand(ctx, cmd); err != nil {
			log.Printf("[enviador] warn: failed to apply command %d (%s): %v", cmd.ID, cmd.Command, err)
			continue
		}
		appliedIDs = append(appliedIDs, cmd.ID)
	}

	if len(appliedIDs) > 0 {
		if err := s.cloudClient.AckPendingCommands(ctx, appliedIDs); err != nil {
			log.Printf("[enviador] warn: failed to ack %d pending commands: %v", len(appliedIDs), err)
		} else {
			log.Printf("[enviador] linea_%d: applied and acked %d pending commands", s.lineaID, len(appliedIDs))
		}
	}
}

// syncMode sincroniza el modo de operación de la línea al cloud si cambió.
// Solo llama al cloud cuando el modo es diferente al último sincronizado.
func (s *SenderService) syncMode(ctx context.Context) {
	mode, err := s.storage.GetCurrentMode(ctx, s.lineaID)
	if err != nil || mode == "" {
		return
	}
	if mode == s.lastSyncedMode {
		return
	}
	if err := s.cloudClient.SyncMode(ctx, s.lineaID, mode); err != nil {
		log.Printf("[enviador] linea_%d: mode sync failed (%s): %v", s.lineaID, mode, err)
		return
	}
	log.Printf("[enviador] linea_%d: mode synced to cloud: %s", s.lineaID, mode)
	s.lastSyncedMode = mode
}

// processEnergy envía un lote de eventos de energía pendientes al cloud.
// Devuelve el número de registros sincronizados; el caller puede usar
//
//	for s.processEnergy(ctx) > 0 {}
//
// para drenar la cola completa en un solo ciclo (flush tras reconexión).
func (s *SenderService) processEnergy(ctx context.Context) int {
	events, err := s.storage.GetUnsyncedEnergyEvents(ctx, s.syncPolicy.GetBatchSize())
	if err != nil {
		log.Printf("[enviador] failed to get unsynced energy events: %v", err)
		return 0
	}
	if len(events) == 0 {
		return 0
	}

	var records []ports.EnergyRecord
	var sentIDs []string // eventos empaquetados correctamente para enviar
	var deadIDs []string // eventos agotados → dead, nunca synced

	for _, event := range events {
		if s.retryPolicy.IsEventExhausted(event.RetryCount) {
			log.Printf("[enviador] energy event %s agotó %d reintentos — marcando dead", event.EventID, event.RetryCount)
			deadIDs = append(deadIDs, event.EventID)
			continue
		}
		rec, err := s.parseEnergyEvent(event)
		if err != nil {
			log.Printf("[enviador] parse energy event %s: %v", event.EventID, err)
			continue
		}
		records = append(records, rec)
		sentIDs = append(sentIDs, event.EventID)
	}

	// Marcar dead los agotados antes de intentar el envío; son distintos
	// de los sincronizados y nunca deben marcarse como synced.
	if len(deadIDs) > 0 {
		if err := s.storage.MarkDead(ctx, deadIDs); err != nil {
			log.Printf("[enviador] warn: no se pudo marcar %d energy events como dead: %v", len(deadIDs), err)
		}
	}

	if len(records) == 0 {
		return 0
	}

	if err := s.cloudClient.SendEnergy(ctx, records); err != nil {
		log.Printf("[enviador] energy send failed: %v", err)
		for _, id := range sentIDs {
			_ = s.storage.IncrementRetry(ctx, id)
		}
		if s.cloudReporter != nil {
			s.cloudReporter.MarkError()
		}
		return 0
	}

	if err := s.storage.MarkSynced(ctx, sentIDs); err != nil {
		log.Printf("[enviador] mark energy synced: %v", err)
		return 0
	}

	if s.cloudReporter != nil {
		s.cloudReporter.MarkCloudOK()
		s.cloudReporter.IncrementSynced(len(records))
	}
	log.Printf("[enviador] synced %d energy snapshots to cloud", len(records))
	return len(records)
}

func (s *SenderService) parseEnergyEvent(event ports.Event) (ports.EnergyRecord, error) {
	var payload struct {
		MeterID   string            `json:"meter_id"`
		IntervalS int               `json:"interval_s"`
		Head      []string          `json:"head"`
		Data      []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return ports.EnergyRecord{}, err
	}

	data := make([]string, len(payload.Data))
	for i, raw := range payload.Data {
		var n json.Number
		if err := json.Unmarshal(raw, &n); err == nil {
			data[i] = n.String()
		} else {
			var s string
			if err := json.Unmarshal(raw, &s); err == nil {
				data[i] = s
			} else {
				data[i] = string(raw)
			}
		}
	}

	return ports.EnergyRecord{
		MeterID:   payload.MeterID,
		Timestamp: event.Timestamp.Unix(), // segundos: el cloud usa time.Unix(ts, 0)
		IntervalS: payload.IntervalS,
		Head:      payload.Head,
		Data:      data,
	}, nil
}
