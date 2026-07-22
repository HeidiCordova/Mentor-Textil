package repository

import (
	"cloud-analytics/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mentor.local/shared/multitenancy"
)

type OEELabRepo struct {
	pr *multitenancy.PoolResolver
}

func NewOEELabRepository(pr *multitenancy.PoolResolver) *OEELabRepo {
	return &OEELabRepo{pr: pr}
}

func (r *OEELabRepo) List(ctx context.Context) ([]domain.OEELabSession, error) {
	pool := r.pr.Master()
	rows, err := pool.Query(ctx, `
		SELECT id, nombre, linea_id, inputs, results, COALESCE(notas,''), COALESCE(created_by,''), created_at, updated_at
		FROM public.oee_lab_sessions
		ORDER BY created_at DESC
		LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("oee_lab list: %w", err)
	}
	defer rows.Close()

	var sessions []domain.OEELabSession
	for rows.Next() {
		var s domain.OEELabSession
		var inputsJSON, resultsJSON []byte
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&s.ID, &s.Nombre, &s.LineaID,
			&inputsJSON, &resultsJSON,
			&s.Notas, &s.CreatedBy,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("oee_lab scan: %w", err)
		}
		s.CreatedAt = createdAt
		s.UpdatedAt = updatedAt
		json.Unmarshal(inputsJSON, &s.Inputs)
		json.Unmarshal(resultsJSON, &s.Results)
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *OEELabRepo) Get(ctx context.Context, id int64) (*domain.OEELabSession, error) {
	pool := r.pr.Master()
	var s domain.OEELabSession
	var inputsJSON, resultsJSON []byte
	var createdAt, updatedAt time.Time
	err := pool.QueryRow(ctx, `
		SELECT id, nombre, linea_id, inputs, results, COALESCE(notas,''), COALESCE(created_by,''), created_at, updated_at
		FROM public.oee_lab_sessions WHERE id = $1`, id).Scan(
		&s.ID, &s.Nombre, &s.LineaID,
		&inputsJSON, &resultsJSON,
		&s.Notas, &s.CreatedBy,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("oee_lab get %d: %w", id, err)
	}
	s.CreatedAt = createdAt
	s.UpdatedAt = updatedAt
	json.Unmarshal(inputsJSON, &s.Inputs)
	json.Unmarshal(resultsJSON, &s.Results)
	return &s, nil
}

func (r *OEELabRepo) Create(ctx context.Context, req *domain.OEELabSessionCreate) (*domain.OEELabSession, error) {
	pool := r.pr.Master()

	inputsJSON, err := json.Marshal(req.Inputs)
	if err != nil {
		return nil, fmt.Errorf("marshal inputs: %w", err)
	}
	resultsJSON, err := json.Marshal(req.Results)
	if err != nil {
		return nil, fmt.Errorf("marshal results: %w", err)
	}

	var id int64
	var createdAt, updatedAt time.Time
	err = pool.QueryRow(ctx, `
		INSERT INTO public.oee_lab_sessions (nombre, linea_id, inputs, results, notas, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		req.Nombre, req.LineaID, inputsJSON, resultsJSON, req.Notas, req.CreatedBy,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("oee_lab insert: %w", err)
	}

	return &domain.OEELabSession{
		ID:        id,
		Nombre:    req.Nombre,
		LineaID:   req.LineaID,
		Inputs:    req.Inputs,
		Results:   req.Results,
		Notas:     req.Notas,
		CreatedBy: req.CreatedBy,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func (r *OEELabRepo) Delete(ctx context.Context, id int64) error {
	pool := r.pr.Master()
	tag, err := pool.Exec(ctx, `DELETE FROM public.oee_lab_sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("oee_lab delete %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("oee_lab session %d not found", id)
	}
	return nil
}
