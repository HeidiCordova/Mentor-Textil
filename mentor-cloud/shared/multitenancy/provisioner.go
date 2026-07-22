package multitenancy

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBProvisioner gestiona la creación de bases de datos y schemas por planta.
// La implementación concreta varía según el entorno (VPS o AWS RDS).
type DBProvisioner interface {
	CreateDatabase(ctx context.Context, dbName string) error
	CreateUser(ctx context.Context, user, password string) error
	GrantAccess(ctx context.Context, user, dbName string) error
	CreateSchema(ctx context.Context, pool *pgxpool.Pool, schema string) error
	RunTemplate(ctx context.Context, pool *pgxpool.Pool, schema string) error
}

// PostgresProvisioner implementa DBProvisioner para entornos VPS con Postgres directo.
type PostgresProvisioner struct {
	masterPool  *pgxpool.Pool
	templateSQL string
}

func NewPostgresProvisioner(masterPool *pgxpool.Pool, templateSQL string) *PostgresProvisioner {
	return &PostgresProvisioner{masterPool: masterPool, templateSQL: templateSQL}
}

func (p *PostgresProvisioner) CreateDatabase(ctx context.Context, dbName string) error {
	_, err := p.masterPool.Exec(ctx, "CREATE DATABASE "+dbName)
	return err
}

func (p *PostgresProvisioner) CreateUser(ctx context.Context, user, password string) error {
	_, err := p.masterPool.Exec(ctx, "CREATE USER "+user+" WITH LOGIN PASSWORD '"+password+"'")
	return err
}

func (p *PostgresProvisioner) GrantAccess(ctx context.Context, user, dbName string) error {
	_, err := p.masterPool.Exec(ctx,
		"GRANT CONNECT ON DATABASE "+dbName+" TO "+user+"; "+
			"REVOKE CONNECT ON DATABASE "+dbName+" FROM PUBLIC",
	)
	return err
}

func (p *PostgresProvisioner) CreateSchema(ctx context.Context, pool *pgxpool.Pool, schema string) error {
	_, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+schema)
	return err
}

func (p *PostgresProvisioner) RunTemplate(ctx context.Context, pool *pgxpool.Pool, schema string) error {
	sql := replaceSchema(p.templateSQL, schema)
	_, err := pool.Exec(ctx, sql)
	return err
}

// RDSProvisioner implementa DBProvisioner para AWS RDS.
type RDSProvisioner struct {
	masterPool  *pgxpool.Pool
	templateSQL string
}

func NewRDSProvisioner(masterPool *pgxpool.Pool, templateSQL string) *RDSProvisioner {
	return &RDSProvisioner{masterPool: masterPool, templateSQL: templateSQL}
}

func (p *RDSProvisioner) CreateDatabase(ctx context.Context, dbName string) error {
	_, err := p.masterPool.Exec(ctx, "CREATE DATABASE "+dbName)
	return err
}

func (p *RDSProvisioner) CreateUser(ctx context.Context, user, password string) error {
	_, err := p.masterPool.Exec(ctx, "CREATE USER "+user+" WITH LOGIN PASSWORD '"+password+"'")
	if err != nil {
		return err
	}
	_, err = p.masterPool.Exec(ctx, "GRANT rds_superuser TO "+user)
	return err
}

func (p *RDSProvisioner) GrantAccess(ctx context.Context, user, dbName string) error {
	_, err := p.masterPool.Exec(ctx, "GRANT CONNECT ON DATABASE "+dbName+" TO "+user)
	return err
}

func (p *RDSProvisioner) CreateSchema(ctx context.Context, pool *pgxpool.Pool, schema string) error {
	_, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+schema)
	return err
}

func (p *RDSProvisioner) RunTemplate(ctx context.Context, pool *pgxpool.Pool, schema string) error {
	sql := replaceSchema(p.templateSQL, schema)
	_, err := pool.Exec(ctx, sql)
	return err
}
