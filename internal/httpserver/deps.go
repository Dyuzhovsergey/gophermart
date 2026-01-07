package httpserver

import (
	"go.uber.org/zap"

	"github.com/Dyuzhovsergey/gophermart/internal/service/auth"
)

type Deps struct {
	Logger *zap.Logger
	Auth   auth.Service
}
