// Package postgres connection via pgxpool.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// LOGIN and PASS
// postgres=# CREATE USER gophermart_dyuzhov WITH PASSWORD 'dyuzhov90419@';

// export DATABASE_URI="postgres://gophermart_dyuzhov:dyuzhov90419%40@localhost:5432/gophermart?sslmode=disable"\

// Подключение к Postgres
// psql "postgres://gophermart_dyuzhov:dyuzhov90419%40@localhost:5432/gophermart?sslmode=disable"

// Connect creates a pgxpool.Pool and verifies connection with Ping.
// ctx is used for both pool creation and ping (recommended: pass ctx with timeout).
func Connect(ctx context.Context, dsn string, log *zap.Logger) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URI is empty")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URI: %w", err)
	}

	// Минимальные sane-defaults для пула (можно будет вынести в конфиг позже)
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	if log != nil {
		log.Info("postgres connected")
	}

	return pool, nil
}
