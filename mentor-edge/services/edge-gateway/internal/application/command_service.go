package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
)

type CommandService struct {
	commands          ports.CommandRepository
	stops             ports.StopRepository
	config            ports.ConfigClient
	audit             ports.AuditRepository
	broker            ports.SSEBroker
	catalogSync       ports.CatalogSyncRepository
	extraCatalogSyncs []ports.CatalogSyncRepository // repos de otras líneas para usuarios globales
	deviceID          string
	localLineaID      int // linea_id local en el edge (ej: 1)
	cloudLineaID      int // linea_id en el cloud (ej: 14)
	wg                sync.WaitGroup
}

func NewCommandService(
	commands ports.CommandRepository,
	stops ports.StopRepository,
	config ports.ConfigClient,
	audit ports.AuditRepository,
	broker ports.SSEBroker,
	catalogSync ports.CatalogSyncRepository,
	deviceID string,
	localLineaID int,
	cloudLineaID int,
) *CommandService {
	return &CommandService{
		commands:     commands,
		stops:        stops,
		config:       config,
		audit:        audit,
		broker:       broker,
		catalogSync:  catalogSync,
		deviceID:     deviceID,
		localLineaID: localLineaID,
		cloudLineaID: cloudLineaID,
	}
}

// SetExtraCatalogSyncs registra los repos de otras líneas para que SYNC_USUARIOS
// escriba a todos los schemas de la planta (usuarios son globales a la planta).
func (s *CommandService) SetExtraCatalogSyncs(repos []ports.CatalogSyncRepository) {
	s.extraCatalogSyncs = repos
}

func (s *CommandService) ReceiveCommand(ctx context.Context, req domain.CreateCommandRequest) (*domain.Command, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if req.DeviceID == "" {
		req.DeviceID = s.deviceID
	}

	cmd, err := s.commands.Create(ctx, req)
	if err == domain.ErrDuplicateCommand {
		existing, _ := s.commands.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		return existing, domain.ErrDuplicateCommand
	}
	if err != nil {
		return nil, err
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.executeCommand(cmd)
	}()

	return cmd, nil
}

func (s *CommandService) Shutdown() {
	s.wg.Wait()
}

func (s *CommandService) GetCommand(ctx context.Context, commandID string) (*domain.Command, error) {
	return s.commands.GetByID(ctx, commandID)
}

func (s *CommandService) ListCommands(ctx context.Context, deviceID string, limit int) ([]domain.Command, error) {
	if deviceID == "" {
		deviceID = s.deviceID
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.commands.ListByDevice(ctx, deviceID, limit)
}

func (s *CommandService) executeCommand(cmd *domain.Command) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result map[string]interface{}
	var execErr error

	switch cmd.CommandType {
	case "CREAR_PARADA":
		result, execErr = s.execCrearParada(ctx, cmd)
	case "MODIFICAR_PARADA":
		result, execErr = s.execModificarParada(ctx, cmd)
	case "JUSTIFICAR_PARADA":
		result, execErr = s.execJustificarParada(ctx, cmd)
	case "CERRAR_PARADA":
		result, execErr = s.execCerrarParada(ctx, cmd)
	case "ELIMINAR_PARADA":
		result, execErr = s.execEliminarParada(ctx, cmd)
	case "ACTUALIZAR_CONFIG":
		result, execErr = s.execActualizarConfig(ctx, cmd)
	case "INICIAR_CALIBRACION":
		result, execErr = s.execIniciarCalibracion(ctx, cmd)
	case "SYNC_CATALOG":
		result, execErr = s.execSyncCatalog(ctx, cmd)
	case "SYNC_PRODUCTOS":
		result, execErr = s.execSyncProductos(ctx, cmd)
	case "SYNC_TURNOS":
		result, execErr = s.execSyncTurnos(ctx, cmd)
	case "SYNC_USUARIOS":
		result, execErr = s.execSyncUsuarios(ctx, cmd)
	case "SYNC_VARIABLES":
		result, execErr = s.execSyncVariables(ctx, cmd)
	case "SYNC_LINEA_PRODUCTO_VARS":
		result, execErr = s.execSyncLineaProductoVars(ctx, cmd)
	case "SYNC_PRODUCTO_CARACTERISTICAS":
		result, execErr = s.execSyncProductoCaracteristicas(ctx, cmd)
	case "SYNC_PLANTAS_LINEAS":
		result, execErr = s.execSyncPlantasLineas(ctx, cmd)
	case "SYNC_PARADAS":
		result, execErr = s.execSyncParadas(ctx, cmd)
	case "SYNC_VELOCIDAD_NOMINAL":
		result, execErr = s.execSyncVelocidadNominal(ctx, cmd)
	case "SYNC_MOTIVOS_VELOCIDAD":
		result, execErr = s.execSyncMotivosVelocidad(ctx, cmd)
	default:
		execErr = fmt.Errorf("unsupported command type: %s", cmd.CommandType)
	}

	if execErr != nil {
		log.Printf("[command] FAILED %s (%s): %v", cmd.CommandID, cmd.CommandType, execErr)
		s.commands.MarkFailed(ctx, cmd.CommandID, execErr.Error())
		s.audit.Log(ctx, domain.AuditEntry{
			DeviceID:   cmd.DeviceID,
			Actor:      cmd.IssuedBy,
			Action:     "EXECUTE_COMMAND",
			Resource:   "command",
			ResourceID: &cmd.CommandID,
			Payload:    cmd.Payload,
			Result:     "ERROR",
		})
		return
	}

	log.Printf("[command] APPLIED %s (%s)", cmd.CommandID, cmd.CommandType)
	s.commands.MarkApplied(ctx, cmd.CommandID, result)
	s.audit.Log(ctx, domain.AuditEntry{
		DeviceID:   cmd.DeviceID,
		Actor:      cmd.IssuedBy,
		Action:     "EXECUTE_COMMAND",
		Resource:   "command",
		ResourceID: &cmd.CommandID,
		Payload:    cmd.Payload,
		Result:     "OK",
	})

	switch cmd.CommandType {
	case "SYNC_CATALOG", "SYNC_PRODUCTOS", "SYNC_TURNOS", "SYNC_VARIABLES",
		"SYNC_LINEA_PRODUCTO_VARS", "SYNC_PRODUCTO_CARACTERISTICAS",
		"SYNC_PLANTAS_LINEAS", "SYNC_VELOCIDAD_NOMINAL", "SYNC_MOTIVOS_VELOCIDAD":
		resource, _ := cmd.Payload["resource"].(string)
		s.broker.Publish("catalogs_synced", map[string]interface{}{"resource": resource})
	}
}

func (s *CommandService) execCrearParada(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	stopType, _ := cmd.Payload["stop_type"].(string)
	if stopType == "" {
		return nil, fmt.Errorf("payload.stop_type is required")
	}

	req := domain.CreateStopRequest{
		DeviceID:  cmd.DeviceID,
		StopType:  stopType,
		StartedAt: time.Now().UTC(),
		Source:    "cloud",
	}

	// Use cloud-generated UUID so both sides share the same stop_id
	if sid, ok := cmd.Payload["stop_id"].(string); ok && sid != "" {
		req.StopID = sid
	}
	// Use cloud-provided timestamp so both sides agree on the start time
	if s, ok := cmd.Payload["started_at"].(string); ok && s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			req.StartedAt = t.UTC()
		}
	}
	if reason, ok := cmd.Payload["reason"].(string); ok {
		req.Reason = &reason
	}
	if category, ok := cmd.Payload["category"].(string); ok {
		req.Category = &category
	}
	if catID, ok := cmd.Payload["categoria_id"].(float64); ok {
		v := int(catID)
		req.CategoriaID = &v
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	stop, err := s.stops.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Broadcast to tablet SSE
	s.broker.Publish("stop_created", map[string]interface{}{
		"stop_id":   stop.StopID,
		"stop_type": stop.StopType,
		"source":    stop.Source,
	})

	return map[string]interface{}{
		"stop_id": stop.StopID,
		"created": true,
	}, nil
}

func (s *CommandService) execModificarParada(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	stopID, _ := cmd.Payload["stop_id"].(string)
	if stopID == "" {
		return nil, fmt.Errorf("payload.stop_id is required")
	}

	stop, err := s.stops.GetByID(ctx, stopID)
	if err != nil {
		return nil, fmt.Errorf("stop not found: %w", err)
	}

	if newType, ok := cmd.Payload["new_type"].(string); ok {
		stop.StopType = newType
	}
	if reason, ok := cmd.Payload["reason"].(string); ok {
		stop.Reason = &reason
	}
	if category, ok := cmd.Payload["category"].(string); ok {
		stop.Category = &category
	}
	if catID, ok := cmd.Payload["categoria_id"].(float64); ok {
		v := int(catID)
		stop.CategoriaID = &v
	}

	if err := s.stops.Update(ctx, stop); err != nil {
		return nil, err
	}

	s.broker.Publish("stop.changed", map[string]interface{}{
		"stop_id": stop.StopID,
	})

	return map[string]interface{}{
		"stop_id":   stop.StopID,
		"stop_type": stop.StopType,
		"updated":   true,
	}, nil
}

func (s *CommandService) execJustificarParada(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	stopID, _ := cmd.Payload["stop_id"].(string)
	reason, _ := cmd.Payload["reason"].(string)
	category, _ := cmd.Payload["category"].(string)
	stopType, _ := cmd.Payload["stop_type"].(string)

	req := domain.JustifyStopRequest{
		StopID:      stopID,
		Reason:      reason,
		Category:    category,
		StopType:    stopType,
		JustifiedBy: cmd.IssuedBy,
	}
	if catID, ok := cmd.Payload["categoria_id"].(float64); ok {
		v := int(catID)
		req.CategoriaID = &v
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	stop, err := s.stops.Justify(ctx, req)
	if err != nil {
		return nil, err
	}

	s.broker.Publish("stop.changed", map[string]interface{}{
		"stop_id":   stop.StopID,
		"justified": true,
	})

	return map[string]interface{}{
		"stop_id":   stop.StopID,
		"justified": true,
	}, nil
}

func (s *CommandService) execCerrarParada(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	stopID, _ := cmd.Payload["stop_id"].(string)
	if stopID == "" {
		return nil, fmt.Errorf("payload.stop_id is required")
	}
	var endedAt interface{}
	if s, ok := cmd.Payload["ended_at"].(string); ok && s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			endedAt = t.UTC()
		}
	}
	stop, err := s.stops.CloseStop(ctx, stopID, endedAt)
	if err != nil {
		return nil, err
	}
	s.broker.Publish("stop_closed", map[string]interface{}{
		"stop_id":  stop.StopID,
		"ended_at": stop.EndedAt,
	})
	return map[string]interface{}{"stop_id": stop.StopID, "closed": true}, nil
}

func (s *CommandService) execEliminarParada(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	stopID, _ := cmd.Payload["stop_id"].(string)
	if stopID == "" {
		return nil, fmt.Errorf("payload.stop_id is required")
	}
	if err := s.stops.Delete(ctx, stopID); err != nil {
		return nil, err
	}
	s.broker.Publish("stop_deleted", map[string]interface{}{"stop_id": stopID})
	return map[string]interface{}{"stop_id": stopID, "deleted": true}, nil
}

func (s *CommandService) execActualizarConfig(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	patch, ok := cmd.Payload["config"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("payload.config must be an object")
	}

	// La configuración OEE (umbrales de microparada/parada) es exclusiva del edge local.
	// Los comandos originados en el cloud no pueden sobreescribir estos valores.
	if isCloudActor(cmd.IssuedBy) {
		delete(patch, "oee")
	}

	result, err := s.config.UpdateConfig(ctx, patch)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"config_updated": true,
		"new_version":    result["config_version"],
	}, nil
}

// isCloudActor devuelve true si el actor que emitió el comando proviene del cloud
// (identificado por el prefijo "cloud:" que asigna el CloudSSEClient en main.go).
func isCloudActor(issuedBy string) bool {
	return len(issuedBy) >= 6 && issuedBy[:6] == "cloud:"
}

func (s *CommandService) execIniciarCalibracion(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	if err := s.config.StartCalibration(ctx); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"calibration_started": true,
	}, nil
}

func (s *CommandService) execSyncCatalog(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.CategoriaParada
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal categoria_paradas: %w", err)
	}
	if err := s.catalogSync.ReplaceCategorias(ctx, records); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"resource": "categoria_paradas",
		"count":    len(records),
		"synced":   true,
	}, nil
}

// translateLineaID convierte cloud linea_id → local linea_id al almacenar syncs.
func (s *CommandService) translateLineaID(id int) int {
	if s.cloudLineaID > 0 && s.localLineaID > 0 && id == s.cloudLineaID {
		return s.localLineaID
	}
	return id
}

func (s *CommandService) execSyncProductos(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.Producto
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal productos: %w", err)
	}
	// Traducir cloud linea_id → local linea_id antes de almacenar
	for i := range records {
		// Forzar el linea_id local si está configurado
		if s.localLineaID > 0 {
			lid := s.localLineaID
			records[i].LineaID = &lid
		} else if records[i].LineaID != nil {
			lid := s.translateLineaID(*records[i].LineaID)
			records[i].LineaID = &lid
		}
	}
	if err := s.catalogSync.ReplaceProductos(ctx, records); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"resource": "productos",
		"count":    len(records),
		"synced":   true,
	}, nil
}

func (s *CommandService) execSyncTurnos(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.Turno
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal turnos: %w", err)
	}
	if err := s.catalogSync.ReplaceTurnos(ctx, records); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"resource": "turnos",
		"count":    len(records),
		"synced":   true,
	}, nil
}

func (s *CommandService) execSyncUsuarios(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.Usuario
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal usuarios: %w", err)
	}
	// Escribir a la línea propia (siempre)
	if err := s.catalogSync.ReplaceUsuarios(ctx, records); err != nil {
		return nil, err
	}
	// Replicar a todas las demás líneas de la planta (usuarios son globales)
	for _, repo := range s.extraCatalogSyncs {
		if err := repo.ReplaceUsuarios(ctx, records); err != nil {
			log.Printf("[execSyncUsuarios] error replicando usuarios a linea extra: %v", err)
		}
	}
	return map[string]interface{}{
		"resource": "usuarios",
		"count":    len(records),
		"synced":   true,
	}, nil
}

func (s *CommandService) execSyncVariables(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.Variable
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal variables: %w", err)
	}
	if err := s.catalogSync.ReplaceVariables(ctx, records); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"resource": "variables",
		"count":    len(records),
		"synced":   true,
	}, nil
}

func (s *CommandService) execSyncLineaProductoVars(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.LineaProductoVar
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal linea_producto_vars: %w", err)
	}
	for i := range records {
		records[i].LineaID = s.translateLineaID(records[i].LineaID)
	}
	if err := s.catalogSync.ReplaceLineaProductoVars(ctx, records); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"resource": "linea_producto_vars",
		"count":    len(records),
		"synced":   true,
	}, nil
}

func (s *CommandService) execSyncProductoCaracteristicas(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.ProductoCaracteristica
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal producto_caracteristicas: %w", err)
	}
	for i := range records {
		records[i].LineaID = s.translateLineaID(records[i].LineaID)
	}
	if err := s.catalogSync.ReplaceProductoCaracteristicas(ctx, records); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"resource": "producto_caracteristicas",
		"count":    len(records),
		"synced":   true,
	}, nil
}

// execSyncPlantasLineas procesa el comando SYNC_PLANTAS_LINEAS.
// Payload: { "records": { "plantas": [...], "lineas": [...] } }
func (s *CommandService) execSyncPlantasLineas(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	recordsData, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var bundle struct {
		Plantas []domain.Planta `json:"plantas"`
		Lineas  []domain.Linea  `json:"lineas"`
	}
	if err := json.Unmarshal(recordsData, &bundle); err != nil {
		return nil, fmt.Errorf("unmarshal plantas_lineas: %w", err)
	}
	if err := s.catalogSync.ReplacePlantas(ctx, bundle.Plantas); err != nil {
		return nil, fmt.Errorf("replace plantas: %w", err)
	}
	if err := s.catalogSync.ReplaceLineas(ctx, bundle.Lineas); err != nil {
		return nil, fmt.Errorf("replace lineas: %w", err)
	}
	return map[string]interface{}{
		"plantas": len(bundle.Plantas),
		"lineas":  len(bundle.Lineas),
		"synced":  true,
	}, nil
}

// execSyncParadas receives an array of cloud stops and upserts them into the
// local edge stops table so the dashboard timeline is always in sync.
func (s *CommandService) execSyncParadas(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}

	type cloudStop struct {
		StopID      string     `json:"stop_id"`
		DeviceID    string     `json:"device_id"`
		StopType    string     `json:"stop_type"`
		StartedAt   time.Time  `json:"started_at"`
		EndedAt     *time.Time `json:"ended_at"`
		Justified   bool       `json:"justified"`
		Reason      *string    `json:"reason"`
		Category    *string    `json:"category"`
		CategoriaID *int       `json:"categoria_id"`
		JustifiedBy *string    `json:"justified_by"`
		JustifiedAt *time.Time `json:"justified_at"`
		Source      string     `json:"source"`
	}

	var records []cloudStop
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal paradas: %w", err)
	}

	upserted := 0
	for _, r := range records {
		if r.StopID == "" {
			continue
		}
		existing, _ := s.stops.GetByID(ctx, r.StopID)
		if existing != nil {
			// Update timing and type from cloud always.
			existing.StopType = r.StopType
			existing.StartedAt = r.StartedAt
			existing.EndedAt = r.EndedAt
			if r.Source != "" {
				existing.Source = r.Source
			}
			// Only overwrite justification fields when the cloud version is justified
			// OR the local stop is not yet justified. This prevents cloud from erasing
			// justifications that the operator set locally but haven't been ACK'd by cloud yet.
			if r.Justified || !existing.Justified {
				existing.Justified = r.Justified
				existing.Reason = r.Reason
				existing.Category = r.Category
				existing.CategoriaID = r.CategoriaID
				existing.JustifiedBy = r.JustifiedBy
				existing.JustifiedAt = r.JustifiedAt
			}
			if err := s.stops.Update(ctx, existing); err != nil {
				log.Printf("[sync-paradas] update %s failed: %v", r.StopID, err)
			} else {
				upserted++
			}
		} else {
			// Create new stop from cloud
			src := r.Source
			if src == "" {
				src = "cloud"
			}
			req := domain.CreateStopRequest{
				StopID:      r.StopID,
				DeviceID:    r.DeviceID,
				StopType:    r.StopType,
				StartedAt:   r.StartedAt,
				EndedAt:     r.EndedAt,
				Reason:      r.Reason,
				Category:    r.Category,
				CategoriaID: r.CategoriaID,
				Source:      src,
			}
			if req.DeviceID == "" {
				req.DeviceID = s.deviceID
			}
			if _, err := s.stops.Create(ctx, req); err != nil {
				log.Printf("[sync-paradas] create %s failed: %v", r.StopID, err)
			} else {
				upserted++
			}
		}
	}

	s.broker.Publish("stops_synced", map[string]interface{}{"count": upserted})
	log.Printf("[sync-paradas] upserted %d/%d paradas", upserted, len(records))

	return map[string]interface{}{
		"resource": "paradas",
		"count":    upserted,
		"total":    len(records),
		"synced":   true,
	}, nil
}

func (s *CommandService) execSyncVelocidadNominal(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.VelocidadNominal
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal velocidad_nominal: %w", err)
	}
	for i := range records {
		records[i].LineaID = s.translateLineaID(records[i].LineaID)
	}
	if err := s.catalogSync.ReplaceVelocidadNominal(ctx, records); err != nil {
		return nil, err
	}
	log.Printf("[sync-velocidad-nominal] synced %d records", len(records))
	return map[string]interface{}{
		"resource": "velocidad_nominal",
		"count":    len(records),
		"synced":   true,
	}, nil
}

func (s *CommandService) execSyncMotivosVelocidad(ctx context.Context, cmd *domain.Command) (map[string]interface{}, error) {
	rawRecords, ok := cmd.Payload["records"]
	if !ok {
		return nil, fmt.Errorf("payload.records is required")
	}
	data, err := json.Marshal(rawRecords)
	if err != nil {
		return nil, fmt.Errorf("marshal records: %w", err)
	}
	var records []domain.MotivoVelocidad
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal motivos_velocidad: %w", err)
	}
	if err := s.catalogSync.ReplaceMotivosVelocidad(ctx, records); err != nil {
		return nil, err
	}
	log.Printf("[sync-motivos-velocidad] synced %d records", len(records))
	return map[string]interface{}{
		"resource": "motivos_velocidad",
		"count":    len(records),
		"synced":   true,
	}, nil
}
