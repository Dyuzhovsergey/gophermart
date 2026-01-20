// Package middleware for work middleware
package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func Logger(l *zap.Logger) func(http.Handler) http.Handler {
	// если логгер не передали — пропускаем
	if l == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w}
			start := time.Now()

			next.ServeHTTP(sw, r)

			l.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", sw.status),
				zap.Int("bytes", sw.bytes),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
			)
		})
	}
}
