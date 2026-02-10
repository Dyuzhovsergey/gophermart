// Package handlers содержит HTTP-хендлеры приложения.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/httperrors"
	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/userctx"
	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/withdrawals"
	"go.uber.org/zap"
)

type WithdrawHandler struct {
	log *zap.Logger
	svc withdrawals.Service
}

type withdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type withdrawalResponseItem struct {
	Order       string  `json:"order"`
	Sum         float64 `json:"sum"`
	ProcessedAt string  `json:"processed_at"`
}

// NewWithdrawHandler создаёт хендлер списаний.
func NewWithdrawHandler(log *zap.Logger, svc withdrawals.Service) *WithdrawHandler {
	return &WithdrawHandler{log: log, svc: svc}
}

// Withdraw — POST /api/user/balance/withdraw.
// Читает JSON, списывает средства с баланса.
func (h *WithdrawHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.UserID(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	req.Order = strings.TrimSpace(req.Order)
	if req.Order == "" || req.Sum <= 0 {
		// По 422 только за неверный номер заказа.
		// Для неверной суммы — 400.
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err := h.svc.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		status := httperrors.MapErrorToStatus(err)
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListWithdrawals — GET /api/user/withdrawals.
func (h *WithdrawHandler) ListWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.UserID(r.Context())
	if !ok {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	items, err := h.svc.List(r.Context(), userID)
	if err != nil {
		status := httperrors.MapErrorToStatus(err)
		http.Error(w, http.StatusText(status), status)
		return
	}

	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent) // 204
		return
	}

	resp := make([]withdrawalResponseItem, 0, len(items)) // список списаний
	for _, it := range items {
		resp = append(resp, withdrawalResponseItem{
			Order:       it.Order,
			Sum:         it.Sum,
			ProcessedAt: it.ProcessedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

var _ = domainerr.ErrNoFunds
