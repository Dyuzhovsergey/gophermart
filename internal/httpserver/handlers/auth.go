package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/httperrors"
	"github.com/Dyuzhovsergey/gophermart/internal/service/auth"
	"go.uber.org/zap"
)

type AuthHandler struct {
	log  *zap.Logger
	auth auth.Service
}

type tokenResponse struct {
	Token string `json:"token"`
}

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// NewAuthHandler создаёт хендлер авторизации.
func NewAuthHandler(log *zap.Logger, authSvc auth.Service) *AuthHandler {
	return &AuthHandler{log: log, auth: authSvc}
}

// Register — POST /api/user/register (JSON login/password).
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Login == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	token, err := h.auth.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		status := httperrors.MapErrorToStatus(err)
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tokenResponse{Token: token})
}

// Login — POST /api/user/login (Basic → JWT в JSON).
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	login, pass, ok := r.BasicAuth()
	if !ok || login == "" || pass == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	token, err := h.auth.LoginBasic(r.Context(), login, pass)
	if err != nil {
		status := httperrors.MapErrorToStatus(err)
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tokenResponse{Token: token})
}
