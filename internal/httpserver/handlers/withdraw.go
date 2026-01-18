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
// Читает JSON {"order": "...", "sum": ...}, списывает средства с баланса.
func (h *WithdrawHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.UserID(r.Context())
	if !ok {
		// На практике сюда не должны попадать, потому что Bearer middleware уже отсекает.
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
		// По ТЗ 422 только за неверный номер заказа.
		// Для неверной суммы — разумно 400.
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err := h.svc.Withdraw(r.Context(), userID, req.Order, req.Sum)
	if err != nil {
		// Если сервис вернул "invalid order" — маппим в 422.
		// Остальные доменные ошибки — через общий маппер.
		status := httperrors.MapErrorToStatus(err)
		http.Error(w, http.StatusText(status), status)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListWithdrawals — GET /api/user/withdrawals.
// Возвращает список списаний или 204, если списаний нет.
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
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]withdrawalResponseItem, 0, len(items))
	for _, it := range items {
		processedAt := it.ProcessedAt
		// На всякий случай: если вдруг нулевое время (не должно быть), ставим текущее.
		if processedAt.IsZero() {
			processedAt = time.Now()
		}

		resp = append(resp, withdrawalResponseItem{
			Order:       it.Order,
			Sum:         it.Sum,
			ProcessedAt: processedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Небольшая "страховка": если где-то ошибочно вернутся не те ошибки.
var _ = domainerr.ErrNoFunds
