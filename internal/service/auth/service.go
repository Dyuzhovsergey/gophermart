// Package auth содержит сервис регистрации и аутентификации.
package auth

import (
	"context"
	"fmt"

	"github.com/Dyuzhovsergey/gophermart/internal/auth/password"
	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/userrepo"
)

// TokenManager — интерфейс для выпуска токена.
type TokenManager interface {
	Generate(userID int64) (string, error)
}

// Service — интерфейс сервиса авторизации.
type Service interface {
	Register(ctx context.Context, login, plainPassword string) (string, error)
	LoginBasic(ctx context.Context, login, plainPassword string) (string, error)
}

type service struct {
	repo userrepo.Repository
	jwt  TokenManager
}

// New создаёт сервис авторизации.
func New(repo userrepo.Repository, jwt TokenManager) Service {
	return &service{repo: repo, jwt: jwt}
}

// Register регистрирует пользователя и возвращает JWT.
// Если логин занят — возвращает domainerr.ErrConflict.
func (s *service) Register(ctx context.Context, login, plainPassword string) (string, error) {
	hash, err := password.HashPassword(plainPassword)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	id, err := s.repo.CreateUser(ctx, login, hash)
	if err != nil {
		return "", err
	}

	token, err := s.jwt.Generate(id)
	if err != nil {
		return "", fmt.Errorf("generate jwt: %w", err)
	}

	return token, nil
}

// LoginBasic аутентифицирует по логину/паролю (из Basic) и возвращает JWT.
// Если пара неверная — возвращает domainerr.ErrUnauthorized.
func (s *service) LoginBasic(ctx context.Context, login, plainPassword string) (string, error) {
	u, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", domainerr.ErrUnauthorized
	}

	ok, err := password.CheckPassword(u.Password, plainPassword)
	if err != nil {
		return "", fmt.Errorf("check password: %w", err)
	}
	if !ok {
		return "", domainerr.ErrUnauthorized
	}

	token, err := s.jwt.Generate(u.ID)
	if err != nil {
		return "", fmt.Errorf("generate jwt: %w", err)
	}

	return token, nil
}
