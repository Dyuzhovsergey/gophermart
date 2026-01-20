package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/httperrors"
	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/userctx"
	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/orders"

	"go.uber.org/zap"
)

type OrdersHandler struct {
	log    *zap.Logger
	orders orders.Service
}

// NewOrdersHandler создаёт хендлер заказов.
func NewOrdersHandler(log *zap.Logger, ordersSvc orders.Service) *OrdersHandler {
	return &OrdersHandler{log: log, orders: ordersSvc}
}

// UploadOrder — POST /api/user/orders (text/plain).
func (h *OrdersHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	number := strings.TrimSpace(string(body))
	if number == "" {
		// Пустой номер заказа.
		http.Error(w, "unprocessable entity", http.StatusUnprocessableEntity)
		return
	}

	err = h.orders.UploadOrder(r.Context(), userID, number)
	if err == nil {
		// Новый номер принят в обработку.
		w.WriteHeader(http.StatusAccepted) // 202
		return
	}

	// Если заказ уже загружен этим пользователем — 200 OK.
	if err == domainerr.ErrAlreadyUploadedByUser {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Все остальные доменные ошибки маппим через единый маппер.
	status := httperrors.MapErrorToStatus(err)
	http.Error(w, http.StatusText(status), status)

	// Лог для диагностики.
	if status >= 500 && h.log != nil {
		h.log.Error("upload order failed", zap.Error(err))
	}
}
