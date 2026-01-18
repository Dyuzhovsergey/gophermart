package httpserver

import (
	"go.uber.org/zap"

	"github.com/Dyuzhovsergey/gophermart/internal/middleware"
	"github.com/Dyuzhovsergey/gophermart/internal/service/accounts"
	"github.com/Dyuzhovsergey/gophermart/internal/service/auth"
	"github.com/Dyuzhovsergey/gophermart/internal/service/orders"
	"github.com/Dyuzhovsergey/gophermart/internal/service/withdrawals"
)

type Deps struct {
	Logger      *zap.Logger
	Auth        auth.Service
	JWT         middleware.TokenVerifier
	Orders      orders.Service
	Accounts    accounts.Service
	Withdrawals withdrawals.Service
}
