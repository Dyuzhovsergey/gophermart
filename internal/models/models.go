// Package models содержит доменные модели проекта.
package models

import "time"

// User — доменная модель пользователя.
type User struct {
	ID        int64     `json:"-"`
	Login     string    `json:"login"`    // login уникальный
	Password  string    `json:"password"` // хеш
	CreatedAt time.Time `json:"-"`
}

// Order — заказ пользователя в системе лояльности.
type Order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// Balance — состояние бонусного счёта пользователя.
type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// Withdrawal — списание средств пользователем.
type Withdrawal struct {
	Order     string    `json:"order"`
	Sum       float64   `json:"sum"`
	Processed time.Time `json:"processed_at"`
}
