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

	return r
}
