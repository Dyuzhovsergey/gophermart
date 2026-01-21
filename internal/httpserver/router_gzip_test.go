package httpserver_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver"
)

// Тест на уровне роутера: middleware.Gzip реально подключён в NewRouter().
func TestRouter_GzipMiddleware_CompressesResponseWhenAccepted(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		// остальные зависимости не нужны для /health
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("want Content-Encoding=gzip, got %q", got)
	}

	// Распаковываем тело и проверяем содержимое.
	gotBody, err := ungzipBytes(w.Body.Bytes())
	if err != nil {
		t.Fatalf("ungzip error: %v", err)
	}
	if string(gotBody) != "OK" {
		t.Fatalf("want body %q, got %q", "OK", string(gotBody))
	}
}

func TestRouter_GzipMiddleware_DoesNotCompressWhenNotAccepted(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// Не ставим Accept-Encoding: gzip

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("want empty Content-Encoding, got %q", got)
	}

	if w.Body.String() != "OK" {
		t.Fatalf("want body %q, got %q", "OK", w.Body.String())
	}
}

func ungzipBytes(b []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}
