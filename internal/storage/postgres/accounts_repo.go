// Package postgres содержит реализацию хранилища на PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountsRepository — репозиторий счетов в Postgres.
type AccountsRepository struct {
	pool *pgxpool.Pool
}

// NewAccountsRepository создаёт репозиторий счетов.
func NewAccountsRepository(pool *pgxpool.Pool) *AccountsRepository {
	return &AccountsRepository{pool: pool}
}

// ProvideAccount гарантирует наличие записи в accounts для пользователя.
func (r *AccountsRepository) ProvideAccount(ctx context.Context, userID int64) error {
	const q = `
		INSERT INTO accounts (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING;
	`
	_, err := r.pool.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("provide account: %w", err)
	}
	return nil
}

// GetBalance возвращает баланс и сумму списаний.
// Если записи нет — возвращает (0, 0, nil).
func (r *AccountsRepository) GetBalance(ctx context.Context, userID int64) (float64, float64, error) {
	const q = `
		SELECT balance::text, withdrawn::text
		FROM accounts
		WHERE user_id = $1;
	`

	var balanceStr string
	var withdrawnStr string

	err := r.pool.QueryRow(ctx, q, userID).Scan(&balanceStr, &withdrawnStr)
	if err == nil {
		b, err := strconv.ParseFloat(balanceStr, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse balance: %w", err)
		}
		w, err := strconv.ParseFloat(withdrawnStr, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse withdrawn: %w", err)
		}
		return b, w, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		// Счёта ещё нет — считаем, что 0/0.
		return 0, 0, nil
	}

	return 0, 0, fmt.Errorf("get balance: %w", err)
}
