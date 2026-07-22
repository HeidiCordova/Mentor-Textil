package repository

import (
	"cloud-config/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PlantaRepo struct {
	db *pgxpool.Pool
}

func NewPlantaRepo(db *pgxpool.Pool) *PlantaRepo {
	return &PlantaRepo{db: db}
}

func (r *PlantaRepo) FindAll(ctx context.Context, empresaID *int) ([]domain.Planta, error) {
	query := `
		SELECT p.id, p.nombre, p.empresa_id, COALESCE(e.nombre,'') AS compania, p.lineas, p.activo, p.tipo,
		       d.db_name, d.instance_type, d.host, d.lineas_creadas, d.provisioned_at,
		       d.rds_class, d.rds_region
		FROM config.plantas p
		LEFT JOIN identity.empresas e ON e.id = p.empresa_id
		LEFT JOIN admin.planta_databases d ON d.planta_id = p.id AND d.provisioned = true`
	args := []interface{}{}

	if empresaID != nil {
		query += ` WHERE p.empresa_id = $1`
		args = append(args, *empresaID)
	}
	query += ` ORDER BY p.id`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Planta
	for rows.Next() {
		var p domain.Planta
		if err := rows.Scan(&p.ID, &p.Nombre, &p.EmpresaID, &p.Compania, &p.Lineas, &p.Activo, &p.Tipo,
			&p.DBName, &p.InstanceType, &p.DBHost, &p.DBSchemas, &p.ProvisionedAt,
			&p.RDSClass, &p.RDSRegion); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *PlantaRepo) FindByID(ctx context.Context, id int) (*domain.Planta, error) {
	p := &domain.Planta{}
	err := r.db.QueryRow(ctx,
		`SELECT p.id, p.nombre, p.empresa_id, COALESCE(e.nombre,'') AS compania, p.lineas, p.activo, p.tipo,
		        d.db_name, d.instance_type, d.host, d.lineas_creadas, d.provisioned_at,
		        d.rds_class, d.rds_region
		 FROM config.plantas p
		 LEFT JOIN identity.empresas e ON e.id = p.empresa_id
		 LEFT JOIN admin.planta_databases d ON d.planta_id = p.id AND d.provisioned = true
		 WHERE p.id=$1`, id,
	).Scan(&p.ID, &p.Nombre, &p.EmpresaID, &p.Compania, &p.Lineas, &p.Activo, &p.Tipo,
		&p.DBName, &p.InstanceType, &p.DBHost, &p.DBSchemas, &p.ProvisionedAt,
		&p.RDSClass, &p.RDSRegion)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PlantaRepo) Create(ctx context.Context, p *domain.Planta) error {
	if p.Tipo == "" {
		p.Tipo = "produccion"
	}
	return r.db.QueryRow(ctx,
		`INSERT INTO config.plantas (nombre, empresa_id, lineas, tipo, activo) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		p.Nombre, p.EmpresaID, p.Lineas, p.Tipo, p.Activo,
	).Scan(&p.ID)
}

func (r *PlantaRepo) Update(ctx context.Context, p *domain.Planta) error {
	if p.Tipo == "" {
		p.Tipo = "produccion"
	}
	_, err := r.db.Exec(ctx,
		`UPDATE config.plantas SET nombre=$2, empresa_id=$3, lineas=$4, tipo=$5, activo=$6, actualizado_en=NOW() WHERE id=$1`,
		p.ID, p.Nombre, p.EmpresaID, p.Lineas, p.Tipo, p.Activo)
	return err
}

func (r *PlantaRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM config.plantas WHERE id=$1`, id)
	return err
}
