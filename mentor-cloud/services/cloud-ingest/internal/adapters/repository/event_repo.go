package repository

import (
	"cloud-ingest/internal/domain"
	"cloud-ingest/internal/ports"
	"context"
	"errors"
	"fmt"
	"strings"

	"mentor.local/shared/multitenancy"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepo struct {
	pr *multitenancy.PoolResolver
}

func NewEventRepo(pr *multitenancy.PoolResolver) *EventRepo {
	return &EventRepo{pr: pr}
}

func (r *EventRepo) resolve(ctx context.Context) (*pgxpool.Pool, string, error) {
	return r.pr.Resolve(ctx)
}

func (r *EventRepo) InsertRawEvent(ctx context.Context, e *domain.RawEvent) error {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "ingest.raw_events")
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO %s (device_id, empresa_id, planta_id, linea_id, event_type, payload, timestamp_edge)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (device_id, timestamp_edge) DO NOTHING
		RETURNING id`, tbl),
		e.DeviceID, e.EmpresaID, e.PlantaID, e.LineaID, e.EventType, e.Payload, e.TimestampEdge,
	).Scan(&e.ID)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func (r *EventRepo) InsertOEESnapshot(ctx context.Context, s *domain.OEESnapshot) error {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	return pool.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO %s
		(device_id, linea_id, planta_id, empresa_id, turno, fecha, hora,
		 disponibilidad, rendimiento, calidad, oee, produccion, energia_kwh,
		 code, interval_s, head, data)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (device_id, hora) DO UPDATE SET
			disponibilidad = EXCLUDED.disponibilidad,
			rendimiento    = EXCLUDED.rendimiento,
			calidad        = EXCLUDED.calidad,
			oee            = EXCLUDED.oee,
			produccion     = EXCLUDED.produccion,
			energia_kwh    = EXCLUDED.energia_kwh,
			head           = EXCLUDED.head,
			data           = EXCLUDED.data
		RETURNING id`, tbl),
		s.DeviceID, s.LineaID, s.PlantaID, s.EmpresaID, s.Turno, s.Fecha, s.Hora,
		s.Disponibilidad, s.Rendimiento, s.Calidad, s.OEE, s.Produccion, s.EnergiaKwh,
		s.Code, s.IntervalS, s.Head, s.Data,
	).Scan(&s.ID)
}

func (r *EventRepo) LogSync(ctx context.Context, l *domain.DeviceSyncLog) error {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "ingest.device_sync_log")
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (device_id, batch_size, status, mensaje)
		VALUES ($1,$2,$3,$4)`, tbl),
		l.DeviceID, l.BatchSize, l.Status, l.Mensaje)
	return err
}

func (r *EventRepo) UpsertParada(ctx context.Context, stop *domain.StopRecord, scope *domain.DeviceScope) error {
	var duracionMin *float64
	if stop.DurationMS != nil {
		v := float64(*stop.DurationMS) / 60000.0
		duracionMin = &v
	}

	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return err
	}

	// Intentar primero con el schema nuevo (tablas separadas por tipo)
	// Si schema está vacío (legacy sin multitenancy) caer en analytics.paradas
	if schema != "" {
		if err := r.upsertParadaSeparada(ctx, pool, schema, stop, scope, duracionMin); err != nil {
			return err
		}
		// Dual-write: también escribir en {schema}.paradas para cloud-analytics
		return r.upsertParadasUnified(ctx, pool, schema, stop, scope, duracionMin)
	}

	tbl := multitenancy.Tbl(schema, "analytics.paradas")
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s
			(device_id, linea_id, planta_id, empresa_id, categoria_id,
			 descripcion, inicio, fin, duracion_min, creado_en,
			 categoria_nombre, stop_id, stop_type, justified, justified_by, justified_at, source, reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::uuid,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (device_id, stop_id) WHERE stop_id IS NOT NULL
		DO UPDATE SET
			fin             = EXCLUDED.fin,
			duracion_min    = EXCLUDED.duracion_min,
			categoria_nombre= EXCLUDED.categoria_nombre,
			stop_type       = EXCLUDED.stop_type,
			-- Si el registro ya está justificado en cloud, conservar datos de cloud.
			-- Si no, aceptar lo que viene del edge.
			categoria_id    = CASE WHEN paradas.justified IS TRUE THEN paradas.categoria_id ELSE EXCLUDED.categoria_id END,
			descripcion     = CASE WHEN paradas.justified IS TRUE THEN paradas.descripcion ELSE EXCLUDED.descripcion END,
			justified       = CASE WHEN paradas.justified IS TRUE THEN paradas.justified ELSE EXCLUDED.justified END,
			justified_by    = CASE WHEN paradas.justified IS TRUE THEN paradas.justified_by ELSE EXCLUDED.justified_by END,
			justified_at    = CASE WHEN paradas.justified IS TRUE THEN paradas.justified_at ELSE EXCLUDED.justified_at END,
			reason          = CASE WHEN paradas.justified IS TRUE THEN paradas.reason ELSE EXCLUDED.reason END`, tbl),
		stop.DeviceID,    // $1
		scope.LineaID,    // $2
		scope.PlantaID,   // $3
		scope.EmpresaID,  // $4
		stop.CategoriaID, // $5
		stop.Reason,      // $6  descripcion
		stop.StartedAt,   // $7  inicio
		stop.EndedAt,     // $8  fin
		duracionMin,      // $9
		stop.CreatedAt,   // $10 creado_en
		stop.Category,    // $11 categoria_nombre
		stop.StopID,      // $12 stop_id UUID
		stop.StopType,    // $13
		stop.Justified,   // $14
		stop.JustifiedBy, // $15
		stop.JustifiedAt, // $16
		stop.Source,      // $17
		stop.Reason,      // $18
	)
	return err
}

// upsertParadaSeparada escribe en parada_programada o parada_no_programada
// según el stop_type. MICROPARADA y PARADA_NO_ASIGNADA se omiten porque se
// acumulan como tiempo en oee_snapshots.head[]/data[].
func (r *EventRepo) upsertParadaSeparada(
	ctx context.Context,
	pool *pgxpool.Pool,
	schema string,
	stop *domain.StopRecord,
	scope *domain.DeviceScope,
	duracionMin *float64,
) error {
	// Determinar tabla destino por stop_type
	var tbl, tblName string
	switch stop.StopType {
	case "PROGRAMADA":
		tblName = "parada_programada"
		tbl = schema + "." + tblName
	case "NO_PROGRAMADA", "MECANICA", "ELECTRICA", "CAMBIO_FORMATO",
		"FALTA_MATERIAL", "CALIDAD", "OTRA":
		tblName = "parada_no_programada"
		tbl = schema + "." + tblName
	default:
		// MICROPARADA, PARADA_NO_ASIGNADA: se capturan en oee_snapshots, no aquí
		return nil
	}

	// Validar que categoria_id existe en el catálogo local; si no, insertar NULL
	catID := stop.CategoriaID
	if catID != nil {
		catTbl := schema + "." + "cat_no_programada"
		if tblName == "parada_programada" {
			catTbl = schema + "." + "cat_programada"
		}
		var exists bool
		_ = pool.QueryRow(ctx, fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE id=$1)`, catTbl), *catID).Scan(&exists)
		if !exists {
			catID = nil
		}
	}

	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s
			(stop_id, device_id, inicio, fin, duracion_min, categoria_id,
			 operador_id, turno, maquina, parte_maquina, descripcion,
			 asignado, asignado_en, creado_en)
		VALUES ($1::uuid,$2,$3,$4,$5,$6,
			NULL, NULL, '', '', COALESCE($7,''),
			$8, $9, $10)
		ON CONFLICT (stop_id) WHERE stop_id IS NOT NULL
		DO UPDATE SET
			fin          = EXCLUDED.fin,
			duracion_min = EXCLUDED.duracion_min,
			-- Si el registro ya está asignado en cloud, conservar datos de cloud.
			-- Si no, aceptar lo que viene del edge.
			categoria_id = CASE WHEN %s.asignado IS TRUE THEN %s.categoria_id ELSE EXCLUDED.categoria_id END,
			descripcion  = CASE WHEN %s.asignado IS TRUE THEN %s.descripcion ELSE EXCLUDED.descripcion END,
			asignado     = CASE WHEN %s.asignado IS TRUE THEN %s.asignado ELSE EXCLUDED.asignado END,
			asignado_en  = CASE WHEN %s.asignado IS TRUE THEN %s.asignado_en ELSE EXCLUDED.asignado_en END`,
		tbl, tblName, tblName, tblName, tblName, tblName, tblName, tblName, tblName),
		stop.StopID,      // $1  stop_id UUID
		stop.DeviceID,    // $2
		stop.StartedAt,   // $3
		stop.EndedAt,     // $4
		duracionMin,      // $5
		catID,            // $6
		stop.Reason,      // $7  descripcion
		stop.Justified,   // $8  asignado
		stop.JustifiedAt, // $9  asignado_en
		stop.CreatedAt,   // $10 creado_en
	)
	return err
}

// upsertParadasUnified escribe en {schema}.paradas con el mismo formato que
// analytics.paradas (usado por cloud-analytics para ListStops/JustifyStop).
func (r *EventRepo) upsertParadasUnified(
	ctx context.Context,
	pool *pgxpool.Pool,
	schema string,
	stop *domain.StopRecord,
	scope *domain.DeviceScope,
	duracionMin *float64,
) error {
	// Ignorar tipos que no tienen stop_id o que se acumulan en oee_snapshots
	if stop.StopID == "" {
		return nil
	}
	// MICROPARADA y PARADA_NO_ASIGNADA sí se sincronizan a paradas para que
	// aparezcan en el timeline de la tablet cloud con ancho proporcional real.

	tbl := schema + ".paradas"
	_, err := pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s
			(device_id, linea_id, planta_id, empresa_id, categoria_id,
			 descripcion, inicio, fin, duracion_min, creado_en,
			 categoria_nombre, stop_id, stop_type, justified, justified_by, justified_at, source, reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::uuid,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (device_id, stop_id) WHERE stop_id IS NOT NULL
		DO UPDATE SET
			fin             = EXCLUDED.fin,
			duracion_min    = EXCLUDED.duracion_min,
			categoria_nombre= EXCLUDED.categoria_nombre,
			stop_type       = EXCLUDED.stop_type,
			-- Preservar datos de justificación si ya fue asignada en cloud
			categoria_id    = CASE WHEN paradas.justified IS TRUE THEN paradas.categoria_id ELSE EXCLUDED.categoria_id END,
			descripcion     = CASE WHEN paradas.justified IS TRUE THEN paradas.descripcion ELSE EXCLUDED.descripcion END,
			justified       = CASE WHEN paradas.justified IS TRUE THEN paradas.justified ELSE EXCLUDED.justified END,
			justified_by    = CASE WHEN paradas.justified IS TRUE THEN paradas.justified_by ELSE EXCLUDED.justified_by END,
			justified_at    = CASE WHEN paradas.justified IS TRUE THEN paradas.justified_at ELSE EXCLUDED.justified_at END,
			reason          = CASE WHEN paradas.justified IS TRUE THEN paradas.reason ELSE EXCLUDED.reason END`, tbl),
		stop.DeviceID,    // $1
		scope.LineaID,    // $2
		scope.PlantaID,   // $3
		scope.EmpresaID,  // $4
		stop.CategoriaID, // $5
		stop.Reason,      // $6  descripcion
		stop.StartedAt,   // $7  inicio
		stop.EndedAt,     // $8  fin
		duracionMin,      // $9
		stop.CreatedAt,   // $10 creado_en
		stop.Category,    // $11 categoria_nombre
		stop.StopID,      // $12 stop_id UUID
		stop.StopType,    // $13
		stop.Justified,   // $14
		stop.JustifiedBy, // $15
		stop.JustifiedAt, // $16
		stop.Source,      // $17
		stop.Reason,      // $18
	)
	return err
}

func (r *EventRepo) GetRawEvents(ctx context.Context, f ports.RawEventsFilter) ([]domain.RawEvent, int, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return nil, 0, err
	}
	tbl := multitenancy.Tbl(schema, "ingest.raw_events")

	// Construir cláusulas WHERE dinámicas
	where := []string{}
	args := []interface{}{}
	idx := 1
	addArg := func(clause string, val interface{}) {
		where = append(where, fmt.Sprintf(clause, idx))
		args = append(args, val)
		idx++
	}

	if f.DeviceID != "" {
		addArg("device_id = $%d", f.DeviceID)
	}
	if f.EmpresaID != nil {
		addArg("empresa_id = $%d", *f.EmpresaID)
	}
	if f.PlantaID != nil {
		addArg("planta_id = $%d", *f.PlantaID)
	}
	if f.LineaID != nil {
		addArg("linea_id = $%d", *f.LineaID)
	}
	if f.EventType != "" {
		addArg("event_type = $%d", f.EventType)
	}
	if f.From != "" {
		addArg("timestamp_edge >= $%d", f.From)
	}
	if f.To != "" {
		addArg("timestamp_edge <= $%d", f.To)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM %s%s`, tbl, whereSQL)
	if err := pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT id, device_id, empresa_id, planta_id, linea_id, event_type, payload, timestamp_edge, recibido_en, procesado
		FROM %s%s ORDER BY recibido_en DESC LIMIT %d OFFSET %d`, tbl, whereSQL, f.Limit, f.Offset)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []domain.RawEvent
	for rows.Next() {
		var e domain.RawEvent
		if err := rows.Scan(&e.ID, &e.DeviceID, &e.EmpresaID, &e.PlantaID, &e.LineaID,
			&e.EventType, &e.Payload, &e.TimestampEdge, &e.RecibidoEn, &e.Procesado); err != nil {
			return nil, 0, err
		}
		result = append(result, e)
	}
	return result, total, nil
}

func (r *EventRepo) GetSnapshots(ctx context.Context, filter ports.OEEFilter) ([]domain.OEESnapshot, error) {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")

	query := fmt.Sprintf(`SELECT id, device_id, linea_id, planta_id, empresa_id, COALESCE(turno,''),
		TO_CHAR(fecha, 'YYYY-MM-DD'), hora,
		COALESCE(disponibilidad,0), COALESCE(rendimiento,0), COALESCE(calidad,0),
		COALESCE(oee,0), COALESCE(produccion,0), COALESCE(energia_kwh,0)
		FROM %s WHERE 1=1`, tbl)
	args := []interface{}{}
	idx := 1

	if filter.DeviceID != "" {
		query += fmt.Sprintf(` AND device_id = $%d`, idx)
		args = append(args, filter.DeviceID)
		idx++
	}
	if filter.PlantaID != nil {
		query += fmt.Sprintf(` AND planta_id = $%d`, idx)
		args = append(args, *filter.PlantaID)
		idx++
	}
	if filter.EmpresaID != nil {
		query += fmt.Sprintf(` AND empresa_id = $%d`, idx)
		args = append(args, *filter.EmpresaID)
		idx++
	}
	if filter.FechaFrom != "" {
		query += fmt.Sprintf(` AND fecha >= $%d`, idx)
		args = append(args, filter.FechaFrom)
		idx++
	}
	if filter.FechaTo != "" {
		query += fmt.Sprintf(` AND fecha <= $%d`, idx)
		args = append(args, filter.FechaTo)
		idx++
	}

	query += ` ORDER BY hora DESC`
	limit := filter.Limit
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	query += fmt.Sprintf(` LIMIT %d`, limit)
	if filter.Offset > 0 {
		query += fmt.Sprintf(` OFFSET %d`, filter.Offset)
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.OEESnapshot
	for rows.Next() {
		var s domain.OEESnapshot
		if err := rows.Scan(&s.ID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
			&s.Turno, &s.Fecha, &s.Hora, &s.Disponibilidad, &s.Rendimiento,
			&s.Calidad, &s.OEE, &s.Produccion, &s.EnergiaKwh); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *EventRepo) GetLatestByDevice(ctx context.Context, deviceID string) (*domain.OEESnapshot, error) {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")

	s := &domain.OEESnapshot{}
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, device_id, linea_id, planta_id, empresa_id, COALESCE(turno,''),
		       TO_CHAR(fecha, 'YYYY-MM-DD'), hora,
		       COALESCE(disponibilidad,0), COALESCE(rendimiento,0), COALESCE(calidad,0),
		       COALESCE(oee,0), COALESCE(produccion,0), COALESCE(energia_kwh,0)
		FROM %s WHERE device_id=$1
		ORDER BY hora DESC LIMIT 1`, tbl), deviceID,
	).Scan(&s.ID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
		&s.Turno, &s.Fecha, &s.Hora, &s.Disponibilidad, &s.Rendimiento,
		&s.Calidad, &s.OEE, &s.Produccion, &s.EnergiaKwh)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *EventRepo) UpdateVariablesFromHeadData(ctx context.Context, deviceID string, head []string, data []string) error {
	if len(head) == 0 || deviceID == "" {
		return nil
	}
	// Las variables en tiempo real siempre viven en mentor_cloud.config.variables (master DB),
	// no en el schema per-planta (linea_14.variables tiene estructura diferente: catálogo de tipos).
	pool := r.pr.Master()
	tblVars := "config.variables"
	tblDisp := "config.dispositivos"

	claves := make([]string, 0, len(head))
	valores := make([]string, 0, len(head))
	for i, clave := range head {
		if i >= len(data) {
			break
		}
		claves = append(claves, clave)
		valores = append(valores, data[i])
	}

	_, err := pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s v
		SET valor = u.valor, actualizado_en = NOW()
		FROM UNNEST($1::text[], $2::text[]) AS u(clave, valor)
		WHERE v.clave = u.clave
		  AND v.dispositivo_id = (
			SELECT id FROM %s WHERE device_id = $3 LIMIT 1
		  )`, tblVars, tblDisp),
		claves, valores, deviceID,
	)
	return err
}

func (r *EventRepo) GetVariablesForEmpresa(ctx context.Context, empresaID int) ([]domain.Variable, error) {
	// Variables en tiempo real siempre en master DB (config.variables).
	pool := r.pr.Master()
	tbl := "config.variables"

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT id, nombre, clave, COALESCE(valor,'0'), COALESCE(tipo,''), dispositivo_id, planta_id, empresa_id, activo
		FROM %s
		WHERE empresa_id = $1
		ORDER BY id`, tbl), empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Variable
	for rows.Next() {
		var v domain.Variable
		if err := rows.Scan(&v.ID, &v.Nombre, &v.Clave, &v.Valor, &v.Tipo, &v.DispositivoID, &v.PlantaID, &v.EmpresaID, &v.Activo); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *EventRepo) UpsertProductionRun(ctx context.Context, run *domain.ProductionRunRecord, scope *domain.DeviceScope) error {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.production_runs")

	if schema != "" {
		// Tabla tenant: ahora incluye linea_id, planta_id, empresa_id (migración 17+)
		_, err = pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s
				(run_id, device_id, linea_id, planta_id, empresa_id,
				 producto_id, sku, nombre, started_at, ended_at,
				 creado_en, actualizado_en)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (device_id, run_id)
			DO UPDATE SET
				ended_at       = EXCLUDED.ended_at,
				producto_id    = EXCLUDED.producto_id,
				sku            = EXCLUDED.sku,
				nombre         = EXCLUDED.nombre,
				actualizado_en = EXCLUDED.actualizado_en`, tbl),
			run.RunID,       // $1
			run.DeviceID,    // $2
			scope.LineaID,   // $3
			scope.PlantaID,  // $4
			scope.EmpresaID, // $5
			run.ProductoID,  // $6
			run.SKU,         // $7
			run.Nombre,      // $8
			run.StartedAt,   // $9
			run.EndedAt,     // $10
			run.CreatedAt,   // $11
			run.UpdatedAt,   // $12
		)
		return err
	}

	// Tabla master: incluye linea_id, planta_id, empresa_id
	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s
			(run_id, device_id, linea_id, planta_id, empresa_id,
			 producto_id, sku, nombre, started_at, ended_at,
			 creado_en, actualizado_en)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (device_id, run_id)
		DO UPDATE SET
			ended_at       = EXCLUDED.ended_at,
			producto_id    = EXCLUDED.producto_id,
			sku            = EXCLUDED.sku,
			nombre         = EXCLUDED.nombre,
			actualizado_en = EXCLUDED.actualizado_en`, tbl),
		run.RunID,       // $1
		run.DeviceID,    // $2
		scope.LineaID,   // $3
		scope.PlantaID,  // $4
		scope.EmpresaID, // $5
		run.ProductoID,  // $6
		run.SKU,         // $7
		run.Nombre,      // $8
		run.StartedAt,   // $9
		run.EndedAt,     // $10
		run.CreatedAt,   // $11
		run.UpdatedAt,   // $12
	)
	return err
}

// GetPendingCommands devuelve los comandos pendientes (no aplicados) para un device_id.
func (r *EventRepo) GetPendingCommands(ctx context.Context, deviceID string) ([]domain.PendingCommand, error) {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.pending_commands")

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT id, device_id, command, payload, applied, created_at
		FROM %s
		WHERE device_id = $1 AND applied = FALSE
		ORDER BY created_at ASC
		LIMIT 100`, tbl), deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.PendingCommand
	for rows.Next() {
		var cmd domain.PendingCommand
		if err := rows.Scan(&cmd.ID, &cmd.DeviceID, &cmd.Command, &cmd.Payload, &cmd.Applied, &cmd.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, cmd)
	}
	return result, nil
}

// AckPendingCommands marca los comandos con los IDs dados como aplicados.
// Filtra por device_id para prevenir que un device marque comandos de otro.
func (r *EventRepo) AckPendingCommands(ctx context.Context, deviceID string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.pending_commands")

	_, err = pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		SET applied = TRUE, applied_at = NOW()
		WHERE device_id = $1 AND id = ANY($2) AND applied = FALSE`, tbl),
		deviceID, ids)
	return err
}

// InsertPendingCommand encola un nuevo comando para ser entregado al edge en el próximo ciclo del Enviador.
func (r *EventRepo) InsertPendingCommand(ctx context.Context, deviceID string, command string, payload []byte) error {
	pool, schema, err := r.resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.pending_commands")

	_, err = pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s (device_id, command, payload)
		VALUES ($1, $2, $3::jsonb)`, tbl),
		deviceID, command, string(payload))
	return err
}
