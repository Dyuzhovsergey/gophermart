// Package password for create hash password and check password.
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrEmptyPassword — передан пустой пароль.
	ErrEmptyPassword = errors.New("empty password")
	// ErrEmptyHash — передан пустой хеш.
	ErrEmptyHash = errors.New("empty hash")
	// ErrWrongPassword — пароль не подходит к хешу.
	ErrWrongPassword = errors.New("wrong password")
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
func CheckPassword(hash, password string) error {
	if hash == "" {
		return ErrEmptyHash
	}
	if password == "" {
		return ErrEmptyPassword
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return nil
	}

	// Неверный пароль — это не "успех", это ожидаемая ошибка.
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrWrongPassword
	}

	// Остальные ошибки считаем проблемой формата/хеша и т.п.
	return fmt.Errorf("compare hash and password: %w", err)
}
