package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

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

type credentialsRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// NewAuthHandler создаёт хендлер авторизации.
func NewAuthHandler(log *zap.Logger, authSvc auth.Service) *AuthHandler {
	return &AuthHandler{log: log, auth: authSvc}
}

// logIfServerError логирует только “нештатные” ошибки (5xx).
// Это помогает находить реальные баги/падения БД/внешних сервисов и т.п.,
// но не засоряет лог ожидаемыми ошибками типа 400/401/409.
func (h *AuthHandler) logIfServerError(r *http.Request, status int, err error, msg string) {
	if status < 500 {
		return
	}
	if h.log == nil {
		return
	}

	h.log.Error(msg,
		zap.Int("status", status),
		zap.Error(err),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote", r.RemoteAddr),
	)
}

// Register — POST /api/user/register (JSON login/password).
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || req.Password == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	token, err := h.auth.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		status := httperrors.MapErrorToStatus(err)
		h.logIfServerError(r, status, err, "register failed")
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tokenResponse{Token: token})
}

// Login — POST /api/user/login (Basic → JWT в JSON).
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var login, pass string

	// 1) Пробуем BasicAuth.
	if u, p, ok := r.BasicAuth(); ok && u != "" && p != "" {
		login, pass = u, p
	} else {
		// 2) Иначе пробуем JSON {login,password}.
		var req credentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		req.Login = strings.TrimSpace(req.Login)
		if req.Login == "" || req.Password == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		login, pass = req.Login, req.Password
	}

	token, err := h.auth.LoginBasic(r.Context(), login, pass)
	if err != nil {
		status := httperrors.MapErrorToStatus(err)
		h.logIfServerError(r, status, err, "login failed")
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tokenResponse{Token: token})
}
