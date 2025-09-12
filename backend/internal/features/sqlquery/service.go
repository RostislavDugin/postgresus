package sqlquery

import (
	"context"
	"errors"
	"time"

	"postgresus-backend/internal/features/databases"
)

type Service struct {
	repo           *Repository
	defaultRows    int
	defaultTimeout time.Duration
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo:           repo,
		defaultRows:    1000,
		defaultTimeout: 30 * time.Second,
	}
}

func (s *Service) Execute(dbc *databases.Database, req *ExecuteRequest) (*ExecuteResponse, error) {
	if req == nil {
		return nil, errors.New("empty request")
	}
	if !IsSafeSelect(req.SQL) {
		return nil, errors.New("only single SELECT/CTE statements are allowed")
	}

	maxRows := req.MaxRows
	if maxRows <= 0 || maxRows > 10000 {
		maxRows = s.defaultRows
	}

	timeout := s.defaultTimeout
	if req.TimeoutSec > 0 && req.TimeoutSec < 120 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res, err := s.repo.ExecuteSelect(ctx, dbc, req.SQL, maxRows)
	if err != nil {
		return nil, err
	}

	return &ExecuteResponse{
		Columns:     res.Columns,
		Rows:        res.Rows,
		RowCount:    res.RowCount,
		Truncated:   res.Truncated,
		ExecutionMs: res.Duration.Milliseconds(),
	}, nil
}
