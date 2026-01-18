package postgres

import (
	"context"
	"fmt"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/withdrawalsrepo"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WithdrawalsRepository struct {
	pool *pgxpool.Pool
}

func NewWithdrawalsRepository(pool *pgxpool.Pool) *WithdrawalsRepository {
	return &WithdrawalsRepository{pool: pool}
}

func (r *WithdrawalsRepository) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1) Блокируем аккаунт
	const qLock = `SELECT balance FROM accounts WHERE user_id = $1 FOR UPDATE;`
	var balance float64
	if err := tx.QueryRow(ctx, qLock, userID).Scan(&balance); err != nil {
		return fmt.Errorf("lock account: %w", err)
	}

	// 2) Проверяем средства
	if balance < sum {
		return domainerr.ErrNoFunds
	}

	// 3) Обновляем счет
	const qUpd = `
		UPDATE accounts
		SET balance = balance - $1,
		    withdrawn = withdrawn + $1
		WHERE user_id = $2;
	`
	if _, err := tx.Exec(ctx, qUpd, sum, userID); err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	// 4) Пишем withdrawals
	const qIns = `
		INSERT INTO withdrawals (user_id, order_number, sum)
		VALUES ($1, $2, $3);
	`
	if _, err := tx.Exec(ctx, qIns, userID, order, sum); err != nil {
		return fmt.Errorf("insert withdrawal: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *WithdrawalsRepository) ListWithdrawals(ctx context.Context, userID int64) ([]withdrawalsrepo.Withdrawal, error) {
	const q = `
		SELECT order_number, sum, processed_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY processed_at DESC;
	`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list withdrawals: %w", err)
	}
	defer rows.Close()

	var out []withdrawalsrepo.Withdrawal
	for rows.Next() {
		var w withdrawalsrepo.Withdrawal
		if err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return out, nil
}
