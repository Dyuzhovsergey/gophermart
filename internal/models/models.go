// Package models
package models

import "time"

// User — доменная модель пользователя
// логин уникальный, пароль хранится в виде хеша
type User struct {
	ID        int64
	Login     string
	Password  string
	CreatedAt time.Time
}

// Order — заказ пользователя в системе лояльности
// status и accrual приходят из внешней системы
type Order struct {
	Number     string
	UserID     int64
	Status     string
	Accrual    float64
	UploadedAt time.Time
}

// Balance — состояние бонусного счета
type Balance struct {
	Current   float64
	Withdrawn float64
}

// Withdrawal — списание средств пользователем
type Withdrawal struct {
	UserID    int64
	Order     string
	Sum       float64
	Processed time.Time
}
