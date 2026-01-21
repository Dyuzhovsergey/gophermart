// Package middleware содержит middleware приложения.
package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Gzip включает gzip-сжатие ответов (если клиент прислал Accept-Encoding: gzip)
// и распаковку входящих запросов (если Content-Encoding: gzip).
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// --- Распаковка входящего тела запроса (если оно gzip) ---
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			// Закрываем исходное тело при завершении запроса.
			origBody := r.Body
			r.Body = &gzipReadCloser{
				Reader: gr,
				closeFn: func() error {
					_ = gr.Close()
					return origBody.Close()
				},
			}
		}

		// --- Сжатие ответа (если клиент поддерживает gzip) ---
		if !clientAcceptsGzip(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Не сжимаем при Upgrade (например websocket).
		if strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") ||
			strings.ToLower(r.Header.Get("Upgrade")) != "" {
			next.ServeHTTP(w, r)
			return
		}

		// Если кто-то уже выставил Content-Encoding — не вмешиваемся.
		if w.Header().Get("Content-Encoding") != "" {
			next.ServeHTTP(w, r)
			return
		}

		gzw := gzip.NewWriter(w)
		defer func() { _ = gzw.Close() }()

		gw := &gzipResponseWriter{
			ResponseWriter: w,
			gw:             gzw,
		}

		// Подсказываем кэшу, что ответ зависит от Accept-Encoding.
		addVaryAcceptEncoding(gw.Header())

		// Выставляем Content-Encoding заранее.
		gw.Header().Set("Content-Encoding", "gzip")

		next.ServeHTTP(gw, r)
	})
}

func clientAcceptsGzip(r *http.Request) bool {
	ae := r.Header.Get("Accept-Encoding")
	return strings.Contains(ae, "gzip")
}

func addVaryAcceptEncoding(h http.Header) {
	const v = "Accept-Encoding"
	cur := h.Get("Vary")
	if cur == "" {
		h.Set("Vary", v)
		return
	}
	// Если уже есть Accept-Encoding — не дублируем.
	parts := strings.Split(cur, ",")
	for _, p := range parts {
		if strings.TrimSpace(strings.ToLower(p)) == strings.ToLower(v) {
			return
		}
	}
	h.Set("Vary", cur+", "+v)
}

// gzipResponseWriter пишет тело ответа в gzip.Writer.
// Важно: некоторые ответы по стандарту не должны иметь тела (204/304) — их не сжимаем.
type gzipResponseWriter struct {
	http.ResponseWriter
	gw          *gzip.Writer
	wroteHeader bool
	statusCode  int
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.wroteHeader = true
	w.statusCode = statusCode

	// 204/304 — без тела: убираем Content-Encoding, чтобы не было некорректного ответа.
	if statusCode == http.StatusNoContent || statusCode == http.StatusNotModified {
		w.Header().Del("Content-Encoding")
		w.ResponseWriter.WriteHeader(statusCode)
		return
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	// Если WriteHeader не вызывали — считаем статус 200.
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	// Если это 204/304 — тело не пишем.
	if w.statusCode == http.StatusNoContent || w.statusCode == http.StatusNotModified {
		return 0, nil
	}
	return w.gw.Write(p)
}

type gzipReadCloser struct {
	io.Reader
	closeFn func() error
}

func (c *gzipReadCloser) Close() error {
	if c.closeFn != nil {
		return c.closeFn()
	}
	return nil
}
