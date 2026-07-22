package handler

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"mentor.local/shared/multitenancy"
)

// resolvePlantaPool resuelve el pool de conexión y nombre de schema para una linea.
func resolvePlantaPool(
	ctx context.Context,
	masterDB *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	lineaID int64,
) (*pgxpool.Pool, string, error) {
	var plantaID int
	if err := masterDB.QueryRow(ctx,
		`SELECT planta_id FROM config.lineas WHERE id = $1`, lineaID,
	).Scan(&plantaID); err != nil {
		return nil, "", fmt.Errorf("linea %d no encontrada: %w", lineaID, err)
	}
	pool, err := mgr.Get(ctx, plantaID)
	if err != nil {
		return nil, "", fmt.Errorf("no se pudo conectar a planta %d: %w", plantaID, err)
	}
	return pool, multitenancy.SchemaName(int(lineaID)), nil
}

func resolvePlantaSchema(
	c *gin.Context,
	masterDB *pgxpool.Pool,
	mgr *multitenancy.PlantaPoolManager,
	lineaID int,
) (*pgxpool.Pool, string, error) {
	return resolvePlantaPool(c.Request.Context(), masterDB, mgr, int64(lineaID))
}
