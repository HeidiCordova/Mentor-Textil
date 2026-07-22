package multitenancy

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolResolver struct {
	master *pgxpool.Pool
	mgr    *PlantaPoolManager
}

func NewPoolResolver(master *pgxpool.Pool, mgr *PlantaPoolManager) *PoolResolver {
	return &PoolResolver{master: master, mgr: mgr}
}

func (r *PoolResolver) Master() *pgxpool.Pool {
	return r.master
}

func (r *PoolResolver) ForPlanta(ctx context.Context, plantaID int) (*pgxpool.Pool, error) {
	if r.mgr == nil {
		return r.master, nil
	}
	return r.mgr.Get(ctx, plantaID)
}

func (r *PoolResolver) Resolve(ctx context.Context) (*pgxpool.Pool, string, error) {
	scope, ok := ScopeFrom(ctx)
	if !ok || r.mgr == nil {
		return r.master, "", nil
	}
	pool, err := r.mgr.Get(ctx, scope.PlantaID)
	if err != nil {
		return nil, "", fmt.Errorf("resolve pool for planta %d: %w", scope.PlantaID, err)
	}
	return pool, SchemaName(scope.LineaID), nil
}
