// Package postgres содержит реализацию хранилища на PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/ordersrepo"
	"github.com/Dyuzhovsergey/gophermart/internal/worker"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrdersRepository — репозиторий заказов в Postgres.
type OrdersRepository struct {
	pool *pgxpool.Pool
}

// NewOrdersRepository создаёт репозиторий заказов.
func NewOrdersRepository(pool *pgxpool.Pool) *OrdersRepository {
	return &OrdersRepository{pool: pool}
}

// AddOrder добавляет заказ с начальным статусом NEW.
// Уникальность обеспечивается PK orders.number.
func (r *OrdersRepository) AddOrder(ctx context.Context, userID int64, number string) error {
	// 1) Быстрая проверка: существует ли заказ и чей он
	const qSel = `
		SELECT user_id
		FROM orders
		WHERE number = $1;
	`

	var existingUserID int64
	err := r.pool.QueryRow(ctx, qSel, number).Scan(&existingUserID)
	if err == nil {
		if existingUserID == userID {
			return domainerr.ErrAlreadyUploadedByUser
		}
		return domainerr.ErrAlreadyUploadedByAnother
	}

	// Если не нашли — пытаемся вставить
	const qIns = `
		INSERT INTO orders (number, user_id, status)
		VALUES ($1, $2, 'NEW');
	`

	_, err = r.pool.Exec(ctx, qIns, number, userID)
	if err == nil {
		return nil
	}

	// Возможна гонка: другой запрос вставил заказ между SELECT и INSERT.
	// Тогда ловим unique violation и повторяем проверку владельца.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		err2 := r.pool.QueryRow(ctx, qSel, number).Scan(&existingUserID)
		if err2 == nil {
			if existingUserID == userID {
				return domainerr.ErrAlreadyUploadedByUser
			}
			return domainerr.ErrAlreadyUploadedByAnother
		}
		return fmt.Errorf("add order after conflict: %w", err2)
	}

	return fmt.Errorf("add order: %w", err)
}

// ListOrdersByUser возвращает заказы пользователя в порядке uploaded_at DESC.
func (r *OrdersRepository) ListOrdersByUser(ctx context.Context, userID int64) ([]ordersrepo.Order, error) {
	const q = `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC;
	`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list orders by user: %w", err)
	}
	defer rows.Close()

	out := make([]ordersrepo.Order, 0)
	for rows.Next() {
		var (
			number     string
			status     string
			accrualStr *string
			uploadedAt time.Time
		)

		// accrual в БД NUMERIC, поэтому безопасно сканим в строку и парсим.
		if err := rows.Scan(&number, &status, &accrualStr, &uploadedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}

		var accrual *float64
		if accrualStr != nil {
			v, err := strconv.ParseFloat(*accrualStr, 64)
			if err != nil {
				return nil, fmt.Errorf("parse accrual: %w", err)
			}
			accrual = &v
		}

		out = append(out, ordersrepo.Order{
			Number:     number,
			Status:     status,
			Accrual:    accrual,
			UploadedAt: uploadedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return out, nil
}

// PickForProcessing выбирает заказы со статусом NEW или PROCESSING.
func (r *OrdersRepository) PickForProcessing(ctx context.Context, limit int) ([]worker.OrderForWork, error) {
	const q = `
		SELECT number
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at ASC
		LIMIT $1;
	`

	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("pick for processing: %w", err)
	}
	defer rows.Close()

	out := make([]worker.OrderForWork, 0)
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		out = append(out, worker.OrderForWork{Number: n})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return out, nil
}

// ApplyAccrualResult обновляет статус заказа и, если нужно, начисляет баллы на счёт.
func (r *OrdersRepository) ApplyAccrualResult(ctx context.Context, number string, status string, accrual *float64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Блокируем строку заказа, чтобы начисление было идемпотентным.
	const qSel = `
		SELECT user_id, accrual
		FROM orders
		WHERE number = $1
		FOR UPDATE;
	`

	var userID int64
	var oldAccrualStr *string
	if err := tx.QueryRow(ctx, qSel, number).Scan(&userID, &oldAccrualStr); err != nil {
		return fmt.Errorf("select order for update: %w", err)
	}

	// Обновляем статус (и accrual, если пришёл).
	// Если accrual == nil, оставляем как есть.
	if accrual == nil {
		const qUpd = `UPDATE orders SET status = $2 WHERE number = $1;`
		if _, err := tx.Exec(ctx, qUpd, number, status); err != nil {
			return fmt.Errorf("update order status: %w", err)
		}
	} else {
		const qUpd = `UPDATE orders SET status = $2, accrual = $3 WHERE number = $1;`
		if _, err := tx.Exec(ctx, qUpd, number, status, *accrual); err != nil {
			return fmt.Errorf("update order status+accrual: %w", err)
		}
	}

	// Начисляем только если:
	// - новый статус PROCESSED
	// - accrual пришёл
	// - в БД ранее accrual был NULL (значит ещё не начисляли)
	if status == "PROCESSED" && accrual != nil && oldAccrualStr == nil {
		const qAcc = `UPDATE accounts SET balance = balance + $1 WHERE user_id = $2;`
		if _, err := tx.Exec(ctx, qAcc, *accrual, userID); err != nil {
			return fmt.Errorf("add balance: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
