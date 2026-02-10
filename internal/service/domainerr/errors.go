// Package domainerr discription domian errors
package domainerr

import "errors"

// ErrBadRequest — некорректные входные данные (пустой логин/пароль и т.п.)
var ErrBadRequest = errors.New("bad request")

// ErrUnauthorized — пользователь не авторизован
var ErrUnauthorized = errors.New("unauthorized")

// ErrConflict — конфликт состояния (например, пользователь уже существует)
var ErrConflict = errors.New("conflict")

// ErrInvalidOrder — неверный номер заказа
var ErrInvalidOrder = errors.New("invalid order")

// ErrNoFunds — недостаточно средств на счете
var ErrNoFunds = errors.New("not enough funds")

// ErrOrderNotFound — заказ не найден
var ErrOrderNotFound = errors.New("order not found")

// ErrAlreadyUploadedByUser — заказ уже загружен этим пользователем
var ErrAlreadyUploadedByUser = errors.New("order already uploaded by this user")

// ErrAlreadyUploadedByAnother — заказ принадлежит другому пользователю
var ErrAlreadyUploadedByAnother = errors.New("order already uploaded by another user")
