// Package password for create hash password and check password.
package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrEmptyPassword — передан пустой пароль.
	ErrEmptyPassword = errors.New("empty password")
	// ErrEmptyHash — передан пустой хеш.
	ErrEmptyHash = errors.New("empty hash")
)

// HashPassword хеширует пароль с использованием bcrypt и возвращает строковый хеш.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// CheckPassword проверяет соответствие пароля bcrypt-хешу.
func CheckPassword(hash, password string) (bool, error) {
	if hash == "" {
		return false, ErrEmptyHash
	}
	if password == "" {
		return false, ErrEmptyPassword
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, nil // пароль подходит
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil // пароль не подходит
	}

	// Остальные ошибки возвращаем как err.
	return false, err
}
