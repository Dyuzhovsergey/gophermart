package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/userctx"
	"github.com/Dyuzhovsergey/gophermart/internal/service/accounts"
	"go.uber.org/zap"
)

type BalanceHandler struct {
	log      *zap.Logger
	accounts accounts.Service
}

// NewBalanceHandler создаёт хендлер баланса.
func NewBalanceHandler(log *zap.Logger, accountsSvc accounts.Service) *BalanceHandler {
	return &BalanceHandler{log: log, accounts: accountsSvc}
}

// GetBalance — GET /api/user/balance.
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	bal, err := h.accounts.GetBalance(r.Context(), userID)
	if err != nil {
		if h.log != nil {
			h.log.Error("get balance failed", zap.Error(err))
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(bal)
}
