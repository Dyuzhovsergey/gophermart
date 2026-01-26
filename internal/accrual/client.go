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

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

// Order — ответ accrual-сервиса.
type Order struct {
	Number  string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

// RateLimitError — типизированная ошибка для 429 Too Many Requests.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("accrual rate limited: retry after %s", e.RetryAfter)
}

// Client — клиент внешнего сервиса начислений.
type Client struct {
	baseURL string
	http    *retryablehttp.Client
}

// New создаёт клиента accrual.
func New(addr string) (*Client, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("accrual address is empty")
	}

	// Добавляем http://
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("parse accrual addr: %w", err)
	}

	rc := retryablehttp.NewClient()

	// Настройки ретраев
	rc.RetryMax = 3
	rc.RetryWaitMin = 100 * time.Millisecond
	rc.RetryWaitMax = 1 * time.Second

	// Отключаем логирование retryablehttp
	rc.Logger = nil

	// Ретраим только сетевые ошибки
	rc.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if resp != nil {
			return false, nil
		}
		return err != nil, nil
	}

	// Таймаут на одну попытку запроса
	rc.HTTPClient.Timeout = 3 * time.Second

	return &Client{
		baseURL: strings.TrimRight(u.String(), "/"),
		http:    rc,
	}, nil
}

// GetOrder получает статус/начисление по заказу.
func (c *Client) GetOrder(ctx context.Context, number string) (*Order, error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/orders/"+number, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		// сетевая ошибка
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var o Order
		if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &o, nil

	case http.StatusNoContent:
		// 204 — заказа ещё нет в accrual
		return nil, nil

	case http.StatusTooManyRequests:
		// 429 — читаем Retry-After (в секундах)
		retryAfter := parseRetryAfter(resp.Header, 60*time.Second)
		return nil, &RateLimitError{RetryAfter: retryAfter}

	default:
		return nil, fmt.Errorf("accrual unexpected status: %d", resp.StatusCode)
	}
}

// parseRetryAfter читает Retry-After из заголовков.
func parseRetryAfter(h http.Header, defaultValue time.Duration) time.Duration {
	ra := strings.TrimSpace(h.Get("Retry-After"))
	if ra == "" {
		return defaultValue
	}

	sec, err := strconv.Atoi(ra)
	if err != nil || sec <= 0 {
		return defaultValue
	}

	return time.Duration(sec) * time.Second
}
