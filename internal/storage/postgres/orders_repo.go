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
