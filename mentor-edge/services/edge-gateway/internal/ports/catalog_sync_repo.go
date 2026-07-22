package ports

import (
	"context"
	"edge-gateway/internal/domain"
)

type CatalogSyncRepository interface {
	ReplaceCategorias(ctx context.Context, records []domain.CategoriaParada) error
	ReplaceProductos(ctx context.Context, records []domain.Producto) error
	ReplaceTurnos(ctx context.Context, records []domain.Turno) error
	ReplaceUsuarios(ctx context.Context, records []domain.Usuario) error
	ReplaceVariables(ctx context.Context, records []domain.Variable) error
	UpdateVariableValor(ctx context.Context, clave string, dispositivo_id int, valor string) error
	ReplaceLineaProductoVars(ctx context.Context, records []domain.LineaProductoVar) error
	ReplaceProductoCaracteristicas(ctx context.Context, records []domain.ProductoCaracteristica) error
	ReplacePlantas(ctx context.Context, records []domain.Planta) error
	ReplaceLineas(ctx context.Context, records []domain.Linea) error
	ReplaceVelocidadNominal(ctx context.Context, records []domain.VelocidadNominal) error
	UpsertVelocidadNominalItems(ctx context.Context, items []domain.VelocidadNominal) error
	InsertVelocidadNominalLog(ctx context.Context, entries []domain.VelocidadNominalLog) error
	ListCategorias(ctx context.Context) ([]domain.CategoriaParada, error)
	ListProductos(ctx context.Context) ([]domain.Producto, error)
	ListTurnos(ctx context.Context) ([]domain.Turno, error)
	ListUsuarios(ctx context.Context) ([]domain.Usuario, error)
	ListVariables(ctx context.Context) ([]domain.Variable, error)
	ListLineaProductoVars(ctx context.Context) ([]domain.LineaProductoVar, error)
	ListProductoCaracteristicas(ctx context.Context) ([]domain.ProductoCaracteristica, error)
	ListPlantas(ctx context.Context) ([]domain.Planta, error)
	ListLineas(ctx context.Context) ([]domain.Linea, error)
	ListVelocidadNominal(ctx context.Context) ([]domain.VelocidadNominal, error)
	ListVelocidadNominalLog(ctx context.Context, limit int) ([]domain.VelocidadNominalLog, error)
	ListMotivosVelocidad(ctx context.Context) ([]domain.MotivoVelocidad, error)
	ReplaceMotivosVelocidad(ctx context.Context, records []domain.MotivoVelocidad) error
}
