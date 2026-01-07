// Package postgres содержит реализацию хранилища на PostgreSQL.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Dyuzhovsergey/gophermart/internal/models"
	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создаёт репозиторий пользователей.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// CreateUser создаёт пользователя в БД и возвращает его idUser.
// Если логин уже занят — возвращает domainerr.ErrConflict.
func (r *UserRepository) CreateUser(ctx context.Context, login, passHash string) (int64, error) {
	const q = `
		INSERT INTO users (login, password)
		VALUES ($1, $2)
		RETURNING id;
	`

	var idUser int64
	err := r.pool.QueryRow(ctx, q, login, passHash).Scan(&idUser)
	if err == nil {
		return idUser, nil
	}

	// Обрабатываем уникальность login (users_login_key или SQLSTATE 23505).
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return 0, domainerr.ErrConflict
	}

	return 0, fmt.Errorf("create user: %w", err)
}

// GetByLogin возвращает пользователя по логину.
// Если пользователь не найден — возвращает (nil, nil).
func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	const q = `
		SELECT id, login, password, created_at
		FROM users
		WHERE login = $1;
	`

	u := &models.User{}
	err := r.pool.QueryRow(ctx, q, login).Scan(&u.ID, &u.Login, &u.Password, &u.CreatedAt)
	if err == nil {
		return u, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	return nil, fmt.Errorf("get user by login: %w", err)
}
