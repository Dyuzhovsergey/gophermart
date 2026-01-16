// Package accrual содержит HTTP-клиент внешней системы начислений.
package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Order — ответ accrual-сервиса.
type Order struct {
	Number  string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

// Client — клиент внешнего сервиса начислений.
type Client struct {
	baseURL string
	http    *http.Client
}

// New создаёт клиента accrual.
// addr может быть вида "localhost:8081" или "http://localhost:8081".
func New(addr string) (*Client, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("accrual address is empty")
	}

	// Если схемы нет — добавим http://
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("parse accrual addr: %w", err)
	}

	return &Client{
		baseURL: strings.TrimRight(u.String(), "/"),
		http: &http.Client{
			Timeout: 3 * time.Second,
		},
	}, nil
}

// GetOrder получает статус/начисление по заказу.
// Возвращает:
// - order != nil при 200
// - order == nil при 204 (заказ не зарегистрирован)
// - retryAfter != nil при 429
func (c *Client) GetOrder(ctx context.Context, number string) (order *Order, retryAfter *time.Duration, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/orders/"+number, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var o Order
		if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
			return nil, nil, fmt.Errorf("decode response: %w", err)
		}
		return &o, nil, nil

	case http.StatusNoContent:
		return nil, nil, nil

	case http.StatusTooManyRequests:
		// Retry-After в секундах (по ТЗ).
		ra := resp.Header.Get("Retry-After")
		if ra == "" {
			d := 60 * time.Second
			return nil, &d, nil
		}
		sec, parseErr := strconv.Atoi(ra)
		if parseErr != nil {
			d := 60 * time.Second
			return nil, &d, nil
		}
		d := time.Duration(sec) * time.Second
		return nil, &d, nil

	default:
		return nil, nil, fmt.Errorf("accrual unexpected status: %d", resp.StatusCode)
	}
}
