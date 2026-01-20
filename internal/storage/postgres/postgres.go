// Package postgres connection via pgxpool.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PoolSettings — настройки пула соединений.
type PoolSettings struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// Connect создает pgxpool.Pool и проверяет соединение с Ping.
func Connect(ctx context.Context, dsn string, log *zap.Logger, ps PoolSettings) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URI is empty")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URI: %w", err)
	}

	cfg.MaxConns = ps.MaxConns
	cfg.MinConns = ps.MinConns
	cfg.MaxConnLifetime = ps.MaxConnLifetime
	cfg.MaxConnIdleTime = ps.MaxConnIdleTime
	cfg.HealthCheckPeriod = ps.HealthCheckPeriod

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
