package application

import (
	"cloud-identity/internal/domain"
	"cloud-identity/internal/ports"
	"context"
)

type EmpresaService struct {
	empresas ports.EmpresaRepository
}

func NewEmpresaService(empresas ports.EmpresaRepository) *EmpresaService {
	return &EmpresaService{empresas: empresas}
}

func (s *EmpresaService) List(ctx context.Context) ([]domain.Empresa, error) {
	return s.empresas.FindAll(ctx)
}

func (s *EmpresaService) Get(ctx context.Context, id int) (*domain.Empresa, error) {
	return s.empresas.FindByID(ctx, id)
}

func (s *EmpresaService) Create(ctx context.Context, e *domain.Empresa) error {
	return s.empresas.Create(ctx, e)
}

func (s *EmpresaService) Update(ctx context.Context, e *domain.Empresa) error {
	return s.empresas.Update(ctx, e)
}

func (s *EmpresaService) Delete(ctx context.Context, id int) error {
	return s.empresas.Delete(ctx, id)
}
