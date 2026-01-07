// Package userrepo содержит интерфейс репозитория пользователей.
package userrepo

import (
	"context"

	"github.com/Dyuzhovsergey/gophermart/internal/models"
)

// Repository — контракт хранилища пользователей.
type Repository interface {
	CreateUser(ctx context.Context, login, passHash string) (int64, error)
	GetByLogin(ctx context.Context, login string) (*models.User, error)
}
