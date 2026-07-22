package ports

import (
	"context"
	"time"
)

type VelocidadNominalRepo interface {
	GetActiveVelocidad(ctx context.Context, lineaID int, ts time.Time) (float64, error)
}
