package repository

import (
	"cloud-ingest/internal/domain"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ScopeResolver struct {
	db *pgxpool.Pool
}

func NewScopeResolver(db *pgxpool.Pool) *ScopeResolver {
	return &ScopeResolver{db: db}
}

func (r *ScopeResolver) ResolveByLinea(ctx context.Context, lineaID int) (*domain.DeviceScope, error) {
	var scope domain.DeviceScope
	scope.LineaID = &lineaID
	err := r.db.QueryRow(ctx,
		`SELECT l.planta_id, p.empresa_id
		   FROM config.lineas l
		   JOIN config.plantas p ON p.id = l.planta_id
		  WHERE l.id = $1`,
		lineaID,
	).Scan(&scope.PlantaID, &scope.EmpresaID)
	if err != nil {
		return nil, fmt.Errorf("scope resolve linea=%d: %w", lineaID, err)
	}
	return &scope, nil
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

func (r *ScopeResolver) UpdateLastSeen(ctx context.Context, deviceID string) {
	r.db.Exec(ctx,
		`UPDATE gateway.device_registry
		    SET last_seen_at = NOW(), active = true
		  WHERE device_id = $1`,
		deviceID,
	)
}
