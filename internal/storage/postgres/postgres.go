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


// Open создаёт пул, проверяет подключение и применяет миграции.
// В результате наружу уходит "готовое к работе" хранилище (без протекающей абстракции).
func Open(ctx context.Context, dsn string, log *zap.Logger, ps PoolSettings) (*pgxpool.Pool, error) {
	pool, err := Connect(ctx, dsn, log, ps)
	if err != nil {
		return nil, err
	}

	// Миграции — часть инициализации хранилища: если они упали, пул закрываем.
	if err := RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	if log != nil {
		log.Info("migrations applied")
	}

	return pool, nil
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

	ps = normalizePoolSettings(ps)

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


// normalizePoolSettings выставляет дефолты, если настройки не заданы.
func normalizePoolSettings(ps PoolSettings) PoolSettings {
	// Важно: нули в конфиге означают "не задано".
	if ps.MaxConns <= 0 {
		ps.MaxConns = 10
	}
	if ps.MinConns < 0 {
		ps.MinConns = 0
	}
	if ps.MaxConnLifetime <= 0 {
		ps.MaxConnLifetime = 30 * time.Minute
	}
	if ps.MaxConnIdleTime <= 0 {
		ps.MaxConnIdleTime = 5 * time.Minute
	}
	if ps.HealthCheckPeriod <= 0 {
		ps.HealthCheckPeriod = 30 * time.Second
	}
	return ps
}