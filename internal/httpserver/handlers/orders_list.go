package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/userctx"
	"github.com/Dyuzhovsergey/gophermart/internal/service/orders"
	"github.com/Dyuzhovsergey/gophermart/internal/service/ordersrepo"
	"go.uber.org/zap"
)

// orderResponse
type orderResponse struct {
	Number     string   `json:"number"`
	Status     string   `json:"status"`
	Accrual    *float64 `json:"accrual,omitempty"`
	UploadedAt string   `json:"uploaded_at"`  // в формате RFC3339.
}

func (h *OrdersHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.UserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.orders.ListOrders(r.Context(), userID)
	if err != nil {
		if h.log != nil {
			h.log.Error("list orders failed", zap.Error(err))
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent) // 204
		return
	}

	resp := make([]orderResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, toOrderResponse(it))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func toOrderResponse(it ordersrepo.Order) orderResponse {
	uploaded := it.UploadedAt.Format(time.RFC3339)

	return orderResponse{
		Number:     it.Number,
		Status:     it.Status,
		Accrual:    it.Accrual,
		UploadedAt: uploaded,
	}
}

// Проверка, что OrdersHandler реализует нужные зависимости.
var _ orders.Service = (orders.Service)(nil)
