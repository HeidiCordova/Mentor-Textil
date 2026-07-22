package repository

import (
	"context"
	"energy-ingest/internal/domain"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ScopeResolver struct {
	db *pgxpool.Pool
}

func NewScopeResolver(db *pgxpool.Pool) *ScopeResolver {
	return &ScopeResolver{db: db}
}

func (r *ScopeResolver) ResolveByDevice(ctx context.Context, deviceID string) (*domain.DeviceScope, error) {
	var scope domain.DeviceScope
	err := r.db.QueryRow(ctx,
		`SELECT empresa_id, planta_id, linea_id
		   FROM gateway.device_registry
		  WHERE device_id = $1 AND active = true`,
		deviceID,
	).Scan(&scope.EmpresaID, &scope.PlantaID, &scope.LineaID)
	if err != nil {
		return nil, fmt.Errorf("scope resolve device=%q: %w", deviceID, err)
	}
	return &scope, nil
}

// ResolveByAPIKey busca en gateway.device_registry por api_key y retorna el
// device_id y scope asociados. Permite autenticación sin X-Device-ID header.
func (r *ScopeResolver) ResolveByAPIKey(ctx context.Context, apiKey string) (string, *domain.DeviceScope, error) {
	var deviceID string
	var scope domain.DeviceScope
	err := r.db.QueryRow(ctx,
		`SELECT device_id, empresa_id, planta_id, linea_id
		   FROM gateway.device_registry
		  WHERE api_key = $1 AND active = true`,
		apiKey,
	).Scan(&deviceID, &scope.EmpresaID, &scope.PlantaID, &scope.LineaID)
	if err != nil {
		return "", nil, fmt.Errorf("resolve api_key: %w", err)
	}
	return deviceID, &scope, nil
}

func (r *ScopeResolver) UpdateLastSeen(ctx context.Context, deviceID string) {
	r.db.Exec(ctx,
		`UPDATE gateway.device_registry
		    SET last_seen_at = NOW(), active = true
		  WHERE device_id = $1`,
		deviceID,
	)
}
