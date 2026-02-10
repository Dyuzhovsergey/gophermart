package postgres

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Встраиваем миграции в бинарь
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations применяет SQL-миграции goose.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Оборачиваем pgxpool в database/sql DB для goose.
	db := stdlib.OpenDBFromPool(pool)

	// Говорим goose читать миграции из embed.FS.
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
