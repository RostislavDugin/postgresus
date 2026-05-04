package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("wal generator exited with error", "error", err)
		os.Exit(1)
	}

	log.Info("shutting down")
}

func run(log *slog.Logger) error {
	pgHost := flag.String("pg-host", "127.0.0.1", "Postgres host")
	pgPort := flag.Int("pg-port", 7433, "Postgres port")
	pgUser := flag.String("pg-user", "devuser", "Postgres user")
	pgPassword := flag.String("pg-password", "devpassword", "Postgres password")
	pgDatabase := flag.String("pg-database", "devdb", "Postgres database")
	rowCount := flag.Int("row-count", 2500, "Rows inserted per cycle (~50MB at default size)")
	cycleSleep := flag.Duration("cycle-sleep", 5*time.Second, "Sleep between cycles")
	tableName := flag.String("table-name", "wal_generator", "Table used for WAL traffic")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		*pgHost, *pgPort, *pgUser, *pgPassword, *pgDatabase,
	)

	conn, err := connectWithRetry(ctx, log, dsn)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(context.Background()); closeErr != nil {
			log.Warn("failed to close postgres connection", "error", closeErr)
		}
	}()

	if err := prepareTable(ctx, conn, *tableName); err != nil {
		return fmt.Errorf("prepare table %q: %w", *tableName, err)
	}

	log.Info(
		fmt.Sprintf("starting wal generation: %d rows per cycle, sleep %s", *rowCount, *cycleSleep),
		"table_name", *tableName,
	)

	return runLoop(ctx, log, conn, *tableName, *rowCount, *cycleSleep)
}

func connectWithRetry(ctx context.Context, log *slog.Logger, dsn string) (*pgx.Conn, error) {
	const retryInterval = 1 * time.Second

	for attempt := 1; ; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		conn, err := pgx.Connect(ctx, dsn)
		if err == nil {
			log.Info(fmt.Sprintf("connected to postgres after %d attempt(s)", attempt))
			return conn, nil
		}

		log.Info(fmt.Sprintf("postgres not ready (attempt %d): %v", attempt, err))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryInterval):
		}
	}
}

func prepareTable(ctx context.Context, conn *pgx.Conn, tableName string) error {
	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", pgx.Identifier{tableName}.Sanitize())
	if _, err := conn.Exec(ctx, dropSQL); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}

	createSQL := fmt.Sprintf(
		"CREATE TABLE %s (id SERIAL PRIMARY KEY, data TEXT NOT NULL)",
		pgx.Identifier{tableName}.Sanitize(),
	)
	if _, err := conn.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	return nil
}

func runLoop(
	ctx context.Context,
	log *slog.Logger,
	conn *pgx.Conn,
	tableName string,
	rowCount int,
	cycleSleep time.Duration,
) error {
	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (data) SELECT repeat(md5(random()::text), 640) FROM generate_series(1, $1)",
		pgx.Identifier{tableName}.Sanitize(),
	)
	deleteSQL := fmt.Sprintf("DELETE FROM %s", pgx.Identifier{tableName}.Sanitize())

	for cycle := 1; ; cycle++ {
		if ctx.Err() != nil {
			return nil
		}

		startedAt := time.Now().UTC()

		if _, err := conn.Exec(ctx, insertSQL, rowCount); err != nil {
			return fmt.Errorf("insert cycle %d: %w", cycle, err)
		}

		if _, err := conn.Exec(ctx, deleteSQL); err != nil {
			return fmt.Errorf("delete cycle %d: %w", cycle, err)
		}

		log.Info(
			fmt.Sprintf("cycle %d completed in %s", cycle, time.Since(startedAt).Round(time.Millisecond)),
			"started_at", startedAt,
		)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(cycleSleep):
		}
	}
}
