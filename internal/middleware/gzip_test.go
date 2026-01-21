package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzip_ResponseCompressedWhenAccepted(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("want Content-Encoding=gzip, got %q", got)
	}

	bodyBytes, err := ungzip(w.Body.Bytes())
	if err != nil {
		t.Fatalf("ungzip error: %v", err)
	}
	if string(bodyBytes) != "hello" {
		t.Fatalf("want body %q, got %q", "hello", string(bodyBytes))
	}
}

func TestGzip_ResponseNotCompressedWhenNotAccepted(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain"))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("want empty Content-Encoding, got %q", got)
	}
	if w.Body.String() != "plain" {
		t.Fatalf("want body %q, got %q", "plain", w.Body.String())
	}
}

func TestGzip_RequestDecompressedWhenEncoded(t *testing.T) {
	// Хендлер читает тело запроса и возвращает его как есть (без gzip-ответа).
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_, _ = w.Write(b)
	}))

	gzBody := gzipBytes([]byte("secret"))
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewReader(gzBody))
	req.Header.Set("Content-Encoding", "gzip")
	// Не просим gzip-ответ, чтобы проще сравнивать.
	req.Header.Set("Accept-Encoding", "identity")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Body.String() != "secret" {
		t.Fatalf("want body %q, got %q", "secret", w.Body.String())
	}
}

func gzipBytes(b []byte) []byte {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

func ungzip(b []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer func() { _ = zr.Close() }()
	return io.ReadAll(zr)
}
