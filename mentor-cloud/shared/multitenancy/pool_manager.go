package multitenancy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"mentor.local/shared/cache"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DSNConfig contiene parámetros de conexión SSL que varían por entorno.
// VPS: SSLMode="disable". RDS: SSLMode="verify-full", SSLRootCert="/certs/rds-ca-root.pem".
type DSNConfig struct {
	SSLMode     string
	SSLRootCert string
}

type plantaEntry struct {
	ID            int
	DBName        string
	PGUser        string
	PGPasswordEnc string
	Host          string
	Port          int
	InstanceType  string
}

// PlantaPoolManager gestiona un pool de conexiones pgx por BD de planta.
// El pool se crea la primera vez que se solicita y se reutiliza indefinidamente.
type PlantaPoolManager struct {
	masterDB    *pgxpool.Pool
	credentials *AESCredentialProvider
	provisioner DBProvisioner
	dsnCfg      DSNConfig
	cache       cache.Store
	pools       sync.Map // planta_id (int) → *pgxpool.Pool
}

func NewPlantaPoolManager(
	masterDB *pgxpool.Pool,
	credentials *AESCredentialProvider,
	provisioner DBProvisioner,
	dsnCfg DSNConfig,
	c cache.Store,
) *PlantaPoolManager {
	return &PlantaPoolManager{
		masterDB:    masterDB,
		credentials: credentials,
		provisioner: provisioner,
		dsnCfg:      dsnCfg,
		cache:       c,
	}
}

// Get retorna el pool de conexiones para la BD de la planta indicada.
// Orden de resolución: sync.Map → CacheStore → admin.planta_databases.
func (m *PlantaPoolManager) Get(ctx context.Context, plantaID int) (*pgxpool.Pool, error) {
	if v, ok := m.pools.Load(plantaID); ok {
		return v.(*pgxpool.Pool), nil
	}

	entry, err := m.resolveEntry(ctx, plantaID)
	if err != nil {
		return nil, err
	}

	password, err := m.credentials.Decrypt(entry.PGPasswordEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials planta %d: %w", plantaID, err)
	}

	dsn := m.buildDSN(entry, password)
	pool, err := m.openPool(dsn)
	if err != nil {
		return nil, fmt.Errorf("open pool planta %d: %w", plantaID, err)
	}

	actual, loaded := m.pools.LoadOrStore(plantaID, pool)
	if loaded {
		pool.Close()
		return actual.(*pgxpool.Pool), nil
	}

	return pool, nil
}

// SchemaName retorna el nombre del schema para una línea dentro de su BD de planta.
func SchemaName(lineaID int) string {
	return fmt.Sprintf("linea_%d", lineaID)
}

func (m *PlantaPoolManager) resolveEntry(ctx context.Context, plantaID int) (*plantaEntry, error) {
	cacheKey := fmt.Sprintf("planta_entry:%d", plantaID)
	if m.cache != nil {
		if raw, ok := m.cache.Get(cacheKey); ok {
			var e plantaEntry
			if json.Unmarshal(raw, &e) == nil {
				return &e, nil
			}
		}
	}

	var entry plantaEntry
	err := m.masterDB.QueryRow(ctx,
		`SELECT id, db_name, pg_user, pg_password_enc, host, port, instance_type
		 FROM admin.planta_databases
		 WHERE planta_id = $1 AND provisioned = true`,
		plantaID,
	).Scan(&entry.ID, &entry.DBName, &entry.PGUser, &entry.PGPasswordEnc,
		&entry.Host, &entry.Port, &entry.InstanceType)
	if err != nil {
		return nil, fmt.Errorf("planta %d not provisioned: %w", plantaID, err)
	}

	if m.cache != nil {
		if b, err := json.Marshal(&entry); err == nil {
			m.cache.Set(cacheKey, b, 10*time.Minute)
		}
	}

	return &entry, nil
}

func (m *PlantaPoolManager) buildDSN(entry *plantaEntry, password string) string {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		entry.Host, entry.Port, entry.DBName, entry.PGUser, password, m.dsnCfg.SSLMode,
	)
	if m.dsnCfg.SSLRootCert != "" {
		dsn += " sslrootcert=" + m.dsnCfg.SSLRootCert
	}
	return dsn
}

func (m *PlantaPoolManager) openPool(dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.MinConns = 2
	config.MaxConns = 10
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// replaceSchema sustituye el placeholder {schema} en el template SQL.
func replaceSchema(sql, schema string) string {
	return strings.ReplaceAll(sql, "{schema}", schema)
}
