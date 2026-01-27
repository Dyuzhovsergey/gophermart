package postgres

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Встраиваем миграции в бинарь, чтобы не зависеть от текущей директории запуска.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations применяет SQL-миграции goose.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	//_ = ctx // goose.Up работает без ctx; оставляем параметр, чтобы не ломать внешний код.
	if err := ctx.Err(); err != nil {
		return err
	}

	// Оборачиваем pgxpool в database/sql DB для goose.
	db := stdlib.OpenDBFromPool(pool) // :contentReference[oaicite:2]{index=2}

	// Говорим goose читать миграции из embed.FS.
	goose.SetBaseFS(migrationsFS) // :contentReference[oaicite:3]{index=3}

	if err := goose.SetDialect("postgres"); err != nil { // :contentReference[oaicite:4]{index=4}
		return fmt.Errorf("goose set dialect: %w", err)
	}

	// Путь "migrations" — это относительный путь ВНУТРИ embed.FS.
	if err := goose.Up(db, "migrations"); err != nil { // :contentReference[oaicite:5]{index=5}
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
