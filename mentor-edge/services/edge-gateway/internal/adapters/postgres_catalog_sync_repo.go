package adapters

import (
	"context"
	"database/sql"
	"edge-gateway/internal/domain"
	"fmt"
	"strings"
	"time"
)

type PostgresCatalogSyncRepo struct {
	db     *sql.DB
	schema string
}

func NewPostgresCatalogSyncRepo(db *sql.DB, schema string) *PostgresCatalogSyncRepo {
	return &PostgresCatalogSyncRepo{db: db, schema: schema}
}

func (r *PostgresCatalogSyncRepo) syncTbl(name string) string { return r.schema + "." + name }

// q reemplaza shared.* con el schema de la línea en todo el SQL.
func (r *PostgresCatalogSyncRepo) q(sql string) string {
	return strings.ReplaceAll(sql, "shared.", r.schema+".")
}

var _ = fmt.Sprintf // keep fmt import

func (r *PostgresCatalogSyncRepo) ReplaceCategorias(ctx context.Context, records []domain.CategoriaParada) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Split by tipo_parada
	var prog, noProg []domain.CategoriaParada
	for _, rec := range records {
		tp := strings.ToUpper(rec.TipoParada)
		if strings.Contains(tp, "NO_PROGRAMADA") || strings.Contains(tp, "NO PROGRAMADA") {
			noProg = append(noProg, rec)
		} else {
			prog = append(prog, rec)
		}
	}

	replaceCatTable := func(table string, items []domain.CategoriaParada) error {
		if _, err := tx.ExecContext(ctx, r.q("DELETE FROM shared."+table)); err != nil {
			return err
		}
		// Pass 1: insert all with padre_id = NULL
		ins, err := tx.PrepareContext(ctx, r.q(
			"INSERT INTO shared."+table+" (id, codigo, nombre, padre_id, orden, activo, creado_en) VALUES ($1,$2,$3,NULL,$4,$5,NOW())"))
		if err != nil {
			return err
		}
		for _, rec := range items {
			if _, err := ins.ExecContext(ctx, rec.ID, rec.Codigo, rec.Nombre, rec.Orden, rec.Activo); err != nil {
				ins.Close()
				return err
			}
		}
		ins.Close()
		// Pass 2: resolve padre_id (NULL if parent not in table)
		upd, err := tx.PrepareContext(ctx, r.q(
			"UPDATE shared."+table+" SET padre_id = (SELECT id FROM shared."+table+" p WHERE p.id = $1) WHERE id = $2"))
		if err != nil {
			return err
		}
		for _, rec := range items {
			if rec.PadreID != nil {
				if _, err := upd.ExecContext(ctx, *rec.PadreID, rec.ID); err != nil {
					upd.Close()
					return err
				}
			}
		}
		upd.Close()
		// Reset sequence past max id
		_, _ = tx.ExecContext(ctx, r.q(
			"SELECT setval(pg_get_serial_sequence('shared."+table+"','id'), COALESCE((SELECT MAX(id) FROM shared."+table+"),0)+1, false)"))
		return nil
	}

	if err := replaceCatTable("cat_programada", prog); err != nil {
		return err
	}
	if err := replaceCatTable("cat_no_programada", noProg); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ReplaceProductos(ctx context.Context, records []domain.Producto) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.productos`)); err != nil {
		return err
	}

	// Forzar linea_id local directamente en el SQL — el schema "linea_N" define N
	localLineaID := r.localLineaIDFromSchema()
	query := r.q(
		`INSERT INTO shared.productos (id, codigo, nombre, empresa_id, activo, linea_id, velocidad_us, factor_conv, synced_at)
		 VALUES ($1, $2, $3, $4, $5, ` + fmt.Sprintf("%d", localLineaID) + `, $6, $7, $8)`)
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.Codigo, rec.Nombre, rec.EmpresaID, rec.Activo, rec.VelocidadUS, rec.FactorConv, now); err != nil {
			return err
		}
	}

	// Safety: forzar linea_id después del INSERT por si el literal no toma efecto
	if localLineaID > 0 {
		if _, err := tx.ExecContext(ctx, r.q(fmt.Sprintf(`UPDATE shared.productos SET linea_id = %d`, localLineaID))); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// localLineaIDFromSchema extrae el número de línea del schema (ej: "linea_1" → 1).
func (r *PostgresCatalogSyncRepo) localLineaIDFromSchema() int {
	// schema es "linea_N" — extraemos N
	var n int
	fmt.Sscanf(r.schema, "linea_%d", &n)
	return n
}

func (r *PostgresCatalogSyncRepo) ReplaceTurnos(ctx context.Context, records []domain.Turno) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.turnos`)); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.turnos (id, nombre, hora_inicio, hora_fin, planta_id, activo, synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.Nombre, rec.HoraInicio, rec.HoraFin, rec.PlantaID, rec.Activo, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ListCategorias(ctx context.Context) ([]domain.CategoriaParada, error) {
	var result []domain.CategoriaParada

	tables := []struct{ name, tipo string }{
		{"cat_programada", "PROGRAMADA"},
		{"cat_no_programada", "NO_PROGRAMADA"},
	}
	for _, tbl := range tables {
		rows, err := r.db.QueryContext(ctx, r.q(
			`SELECT id, nombre, COALESCE(codigo,''), padre_id, orden, activo, creado_en
			 FROM shared.`+tbl.name+` ORDER BY id`))
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var c domain.CategoriaParada
			if err := rows.Scan(&c.ID, &c.Nombre, &c.Codigo, &c.PadreID, &c.Orden, &c.Activo, &c.SyncedAt); err != nil {
				rows.Close()
				return nil, err
			}
			c.TipoParada = tbl.tipo
			result = append(result, c)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *PostgresCatalogSyncRepo) ListProductos(ctx context.Context) ([]domain.Producto, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, codigo, nombre, empresa_id, activo, linea_id, velocidad_us, factor_conv, synced_at FROM shared.productos ORDER BY id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Producto
	for rows.Next() {
		var p domain.Producto
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Nombre, &p.EmpresaID, &p.Activo, &p.LineaID, &p.VelocidadUS, &p.FactorConv, &p.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ListTurnos(ctx context.Context) ([]domain.Turno, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, nombre, hora_inicio, hora_fin, planta_id, activo, synced_at FROM shared.turnos ORDER BY id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Turno
	for rows.Next() {
		var t domain.Turno
		if err := rows.Scan(&t.ID, &t.Nombre, &t.HoraInicio, &t.HoraFin, &t.PlantaID, &t.Activo, &t.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ReplaceUsuarios(ctx context.Context, records []domain.Usuario) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.usuarios`)); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.usuarios (id, username, email, nombre, apellido, password_hash, rol_id, rol, empresa_id, activo, synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.Username, rec.Email, rec.Nombre, rec.Apellido, rec.PasswordHash, rec.RolID, rec.Rol, rec.EmpresaID, rec.Activo, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ReplaceVariables(ctx context.Context, records []domain.Variable) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.variables`)); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.variables (id, nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo, synced_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.Nombre, rec.Clave, rec.Valor, rec.Tipo, rec.DispositivoID, rec.PlantaID, rec.EmpresaID, rec.Activo, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpdateVariableValor actualiza el campo valor de una variable local (por clave + dispositivo_id).
// Si la variable no existe la crea (upsert por software: UPDATE → INSERT si 0 filas afectadas).
// Si dispositivo_id <= 0 se opera por clave solamente (primer match).
func (r *PostgresCatalogSyncRepo) UpdateVariableValor(ctx context.Context, clave string, dispositivoID int, valor string) error {
	var res sql.Result
	var err error
	if dispositivoID > 0 {
		res, err = r.db.ExecContext(ctx, r.q(
			`UPDATE shared.variables SET valor = $1 WHERE clave = $2 AND dispositivo_id = $3`),
			valor, clave, dispositivoID)
	} else {
		res, err = r.db.ExecContext(ctx, r.q(
			`UPDATE shared.variables SET valor = $1 WHERE clave = $2`),
			valor, clave)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// La variable no existe aún: insertarla con un ID generado localmente.
		_, err = r.db.ExecContext(ctx, r.q(
			`INSERT INTO shared.variables (id, nombre, clave, valor, tipo, activo, synced_at)
			 VALUES (COALESCE((SELECT MAX(id)+1 FROM shared.variables), 1), $1, $2, $3, 'number', true, NOW())`),
			clave, clave, valor)
	}
	return err
}

func (r *PostgresCatalogSyncRepo) ListUsuarios(ctx context.Context) ([]domain.Usuario, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, username, COALESCE(email,''), nombre, rol_id, rol, empresa_id, activo, synced_at FROM shared.usuarios ORDER BY id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Usuario
	for rows.Next() {
		var u domain.Usuario
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Nombre, &u.RolID, &u.Rol, &u.EmpresaID, &u.Activo, &u.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ListVariables(ctx context.Context) ([]domain.Variable, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo, synced_at FROM shared.variables ORDER BY id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Variable
	for rows.Next() {
		var v domain.Variable
		if err := rows.Scan(&v.ID, &v.Nombre, &v.Clave, &v.Valor, &v.Tipo, &v.DispositivoID, &v.PlantaID, &v.EmpresaID, &v.Activo, &v.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ReplaceLineaProductoVars(ctx context.Context, records []domain.LineaProductoVar) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.linea_producto_vars`)); err != nil {
		return err
	}

	localLineaID := r.localLineaIDFromSchema()
	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.linea_producto_vars (id, linea_id, variable_id, nombre_col, orden, synced_at)
		 VALUES ($1, `+fmt.Sprintf("%d", localLineaID)+`, $2, $3, $4, $5)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.VariableID, rec.NombreCol, rec.Orden, now); err != nil {
			return err
		}
	}

	if localLineaID > 0 {
		if _, err := tx.ExecContext(ctx, r.q(fmt.Sprintf(`UPDATE shared.linea_producto_vars SET linea_id = %d`, localLineaID))); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ReplaceProductoCaracteristicas(ctx context.Context, records []domain.ProductoCaracteristica) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.producto_caracteristicas`)); err != nil {
		return err
	}

	localLineaID := r.localLineaIDFromSchema()
	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.producto_caracteristicas (id, producto_id, linea_id, variable_id, valor, synced_at)
		 VALUES ($1, $2, `+fmt.Sprintf("%d", localLineaID)+`, $3, $4, $5)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.ProductoID, rec.VariableID, rec.Valor, now); err != nil {
			return err
		}
	}

	if localLineaID > 0 {
		if _, err := tx.ExecContext(ctx, r.q(fmt.Sprintf(`UPDATE shared.producto_caracteristicas SET linea_id = %d`, localLineaID))); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ListLineaProductoVars(ctx context.Context) ([]domain.LineaProductoVar, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, linea_id, variable_id, nombre_col, orden, synced_at FROM shared.linea_producto_vars ORDER BY orden`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.LineaProductoVar
	for rows.Next() {
		var v domain.LineaProductoVar
		if err := rows.Scan(&v.ID, &v.LineaID, &v.VariableID, &v.NombreCol, &v.Orden, &v.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ListProductoCaracteristicas(ctx context.Context) ([]domain.ProductoCaracteristica, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, producto_id, linea_id, variable_id, valor, synced_at FROM shared.producto_caracteristicas ORDER BY producto_id, variable_id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProductoCaracteristica
	for rows.Next() {
		var c domain.ProductoCaracteristica
		if err := rows.Scan(&c.ID, &c.ProductoID, &c.LineaID, &c.VariableID, &c.Valor, &c.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ReplacePlantas(ctx context.Context, records []domain.Planta) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.plantas`)); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.plantas (id, nombre, empresa_id, empresa_nombre, activo, synced_at) VALUES ($1, $2, $3, $4, $5, $6)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.Nombre, rec.EmpresaID, rec.EmpresaNombre, rec.Activo, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ReplaceLineas(ctx context.Context, records []domain.Linea) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.lineas`)); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.lineas (id, nombre, planta_id, tipo, subtipo, activo, synced_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ID, rec.Nombre, rec.PlantaID, rec.Tipo, rec.Subtipo, rec.Activo, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ListPlantas(ctx context.Context) ([]domain.Planta, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, nombre, empresa_id, empresa_nombre, activo, synced_at FROM shared.plantas ORDER BY id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Planta
	for rows.Next() {
		var p domain.Planta
		if err := rows.Scan(&p.ID, &p.Nombre, &p.EmpresaID, &p.EmpresaNombre, &p.Activo, &p.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ListLineas(ctx context.Context) ([]domain.Linea, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, nombre, planta_id, tipo, subtipo, activo, synced_at FROM shared.lineas ORDER BY id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Linea
	for rows.Next() {
		var l domain.Linea
		if err := rows.Scan(&l.ID, &l.Nombre, &l.PlantaID, &l.Tipo, &l.Subtipo, &l.Activo, &l.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ReplaceVelocidadNominal(ctx context.Context, records []domain.VelocidadNominal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.velocidad_nominal`)); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.velocidad_nominal (linea_id, producto_id, velocidad_us, factor_conv, synced_at)
		 VALUES (`+fmt.Sprintf("%d", r.localLineaIDFromSchema())+`, $1, $2, $3, $4)`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, rec := range records {
		if _, err := stmt.ExecContext(ctx, rec.ProductoID, rec.VelocidadUs, rec.FactorConv, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) UpsertVelocidadNominalItems(ctx context.Context, items []domain.VelocidadNominal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	localLineaID := r.localLineaIDFromSchema()
	stmt, err := tx.PrepareContext(ctx, r.q(`
		INSERT INTO shared.velocidad_nominal (linea_id, producto_id, velocidad_us, factor_conv, synced_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (linea_id, producto_id)
		DO UPDATE SET
			velocidad_us = EXCLUDED.velocidad_us,
			factor_conv  = EXCLUDED.factor_conv,
			synced_at    = NOW()`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		fc := item.FactorConv
		if fc <= 0 {
			fc = 1
		}
		if _, err := stmt.ExecContext(ctx, localLineaID, item.ProductoID, item.VelocidadUs, fc); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresCatalogSyncRepo) ListVelocidadNominal(ctx context.Context) ([]domain.VelocidadNominal, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, linea_id, producto_id, velocidad_us, factor_conv, synced_at
		 FROM shared.velocidad_nominal ORDER BY linea_id, producto_id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.VelocidadNominal
	for rows.Next() {
		var v domain.VelocidadNominal
		if err := rows.Scan(&v.ID, &v.LineaID, &v.ProductoID, &v.VelocidadUs, &v.FactorConv, &v.SyncedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) InsertVelocidadNominalLog(ctx context.Context, entries []domain.VelocidadNominalLog) error {
	if len(entries) == 0 {
		return nil
	}
	stmt, err := r.db.PrepareContext(ctx, r.q(`
		INSERT INTO shared.velocidad_nominal_log
			(producto_id, sku, velocidad_us_anterior, velocidad_us_nueva,
			 factor_conv_anterior, factor_conv_nueva, motivo, usuario, origen, cambiado_en)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())`))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.ExecContext(ctx,
			e.ProductoID, e.SKU,
			e.VelocidadUSAnterior, e.VelocidadUSNueva,
			e.FactorConvAnterior, e.FactorConvNueva,
			e.Motivo, e.Usuario, e.Origen,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresCatalogSyncRepo) ListVelocidadNominalLog(ctx context.Context, limit int) ([]domain.VelocidadNominalLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, r.q(`
		SELECT id, producto_id, sku,
		       velocidad_us_anterior, velocidad_us_nueva,
		       factor_conv_anterior, factor_conv_nueva,
		       motivo, usuario, origen, cambiado_en
		FROM shared.velocidad_nominal_log
		ORDER BY cambiado_en DESC
		LIMIT $1`), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.VelocidadNominalLog
	for rows.Next() {
		var e domain.VelocidadNominalLog
		if err := rows.Scan(
			&e.ID, &e.ProductoID, &e.SKU,
			&e.VelocidadUSAnterior, &e.VelocidadUSNueva,
			&e.FactorConvAnterior, &e.FactorConvNueva,
			&e.Motivo, &e.Usuario, &e.Origen, &e.CambiadoEn,
		); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ListMotivosVelocidad(ctx context.Context) ([]domain.MotivoVelocidad, error) {
	rows, err := r.db.QueryContext(ctx, r.q(
		`SELECT id, texto, activo, orden FROM shared.motivos_velocidad ORDER BY orden, id`))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.MotivoVelocidad
	for rows.Next() {
		var m domain.MotivoVelocidad
		if err := rows.Scan(&m.ID, &m.Texto, &m.Activo, &m.Orden); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *PostgresCatalogSyncRepo) ReplaceMotivosVelocidad(ctx context.Context, records []domain.MotivoVelocidad) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, r.q(`DELETE FROM shared.motivos_velocidad`)); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, r.q(
		`INSERT INTO shared.motivos_velocidad (id, texto, activo, orden) VALUES ($1, $2, $3, $4)`))
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, m := range records {
		if _, err := stmt.ExecContext(ctx, m.ID, m.Texto, m.Activo, m.Orden); err != nil {
			return err
		}
	}
	return tx.Commit()
}
