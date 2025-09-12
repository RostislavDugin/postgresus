package sqlquery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"postgresus-backend/internal/features/databases"
	"postgresus-backend/internal/features/databases/databases/postgresql"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

func (r *Repository) openPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 2
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 90 * time.Second
	return pgxpool.NewWithConfig(ctx, cfg)
}

func buildPostgresDSN(pg *postgresql.PostgresqlDatabase) (string, error) {
	if pg == nil {
		return "", fmt.Errorf("postgres config is nil")
	}
	if pg.Database == nil {
		return "", fmt.Errorf("database name is nil")
	}

	db := *pg.Database

	ssl := "disable"
	if pg.IsHttps {
		ssl = "require"
	}

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pg.Host,
		pg.Port,
		pg.Username,
		pg.Password,
		db,
		ssl,
	), nil
}

type Result struct {
	Columns   []string
	Rows      [][]any
	RowCount  int
	Truncated bool
	Duration  time.Duration
}

// ExecuteSelect — safe SELECT/CTE with forced LIMIT
func (r *Repository) ExecuteSelect(ctx context.Context, dbc *databases.Database, sql string, maxRows int) (*Result, error) {
	if dbc == nil {
		return nil, fmt.Errorf("database config is nil")
	}
	t := strings.ToUpper(string(dbc.Type))
	if t != "POSTGRES" && t != "POSTGRESQL" {
		return nil, fmt.Errorf("only PostgreSQL type is supported, got %q", dbc.Type)
	}

	dsn, err := buildPostgresDSN(dbc.Postgresql)
	if err != nil {
		return nil, err
	}

	sql = ensureLimit(sql, maxRows)

	pool, err := r.openPool(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	start := time.Now()
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	cols := make([]string, len(fds))
	for i, fd := range fds {
		cols[i] = fd.Name
	}

	outRows := make([][]any, 0, 64)
	rowCount := 0
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		out := make([]any, len(vals))
		copy(out, vals)
		outRows = append(outRows, out)
		rowCount++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	dur := time.Since(start)
	trunc := rowCount >= maxRows

	return &Result{
		Columns:   cols,
		Rows:      outRows,
		RowCount:  rowCount,
		Truncated: trunc,
		Duration:  dur,
	}, nil
}

// ensureLimit — guaranteed to wrap in a subscript with LIMIT
func ensureLimit(sql string, maxRows int) string {
	s := strings.TrimSpace(sql)
	if maxRows <= 0 {
		maxRows = 1000
	}
	return fmt.Sprintf("SELECT * FROM (%s) __q LIMIT %d", s, maxRows)
}
