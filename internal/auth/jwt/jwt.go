// Package jwt for generate and check JWT-token (HS256).
package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

var (
	// ErrEmptySecret — пустой секрет для подписи JWT.
	ErrEmptySecret = errors.New("empty jwt secret")
	// ErrEmptyToken — пустая строка токена.
	ErrEmptyToken = errors.New("empty jwt token")
	// ErrInvalidToken — невалидный токен (формат/подпись/claims).
	ErrInvalidToken = errors.New("invalid jwt token")
	// ErrExpiredToken — токен просрочен.
	ErrExpiredToken = errors.New("jwt token expired")
)

// Manager управляет выпуском и проверкой JWT-токенов.
type Manager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// New создаёт менеджер JWT.
// secret — ключ подписи HS256, ttl — время жизни токена (например 24h).
func New(secret string, ttl time.Duration) (*Manager, error) {
	if secret == "" {
		return nil, ErrEmptySecret
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be positive")
	}

	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}, nil
}

// Generate выпускает JWT для пользователя.
// userID кладём в standard claim "sub" (subject).
func (m *Manager) Generate(userID int64) (string, error) {
	now := m.now()

	claims := jwtlib.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwtlib.NewNumericDate(now),
		ExpiresAt: jwtlib.NewNumericDate(now.Add(m.ttl)),
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Verify проверяет токен и возвращает userID из claim "sub".
func (m *Manager) Verify(tokenString string) (int64, error) {
	if tokenString == "" {
		return 0, ErrEmptyToken
	}

	claims := &jwtlib.RegisteredClaims{}

	// разрешаем только HS256
	token, err := jwtlib.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwtlib.Token) (any, error) {
			if t.Method != jwtlib.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return m.secret, nil
		},
		jwtlib.WithValidMethods([]string{jwtlib.SigningMethodHS256.Alg()}),
		jwtlib.WithTimeFunc(m.now),
	)
	if err != nil {
		// Просрочка токена — отдельный кейс
		if errors.Is(err, jwtlib.ErrTokenExpired) {
			return 0, ErrExpiredToken
		}
		return 0, ErrInvalidToken
	}

	if !token.Valid {
		return 0, ErrInvalidToken
	}

	if claims.Subject == "" {
		return 0, ErrInvalidToken
	}

	idUser, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidToken
	}

	return idUser, nil
}
