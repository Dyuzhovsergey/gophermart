// Package httpserver for work HTTP server
package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/handlers"
	"github.com/Dyuzhovsergey/gophermart/internal/middleware"
)

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recover(deps.Logger))
	r.Use(middleware.Logger(deps.Logger))

	r.Get("/health", handlers.Health)

	// AUTH
	authHandler := handlers.NewAuthHandler(deps.Logger, deps.Auth)
	r.Post("/api/user/register", authHandler.Register)
	r.Post("/api/user/login", authHandler.Login)

	// Защищённая зона /api/user/*
	r.Route("/api/user", func(sr chi.Router) {
		sr.Use(middleware.BearerAuth(deps.JWT))

		if deps.Orders != nil {
			ordersHandler := handlers.NewOrdersHandler(deps.Logger, deps.Orders)
			sr.Post("/orders", ordersHandler.UploadOrder)
			sr.Get("/orders", ordersHandler.ListOrders)
		}

		if deps.Accounts != nil {
			balanceHandler := handlers.NewBalanceHandler(deps.Logger, deps.Accounts)
			sr.Get("/balance", balanceHandler.GetBalance)
		}

		// sr.Post("/orders", ...) +
		// sr.Get("/orders", ...) +
		// sr.Get("/balance", ...) +
		// sr.Post("/balance/withdraw", ...)
		// sr.Get("/withdrawals", ...)
	})

	return r
}
