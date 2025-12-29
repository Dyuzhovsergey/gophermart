package middleware

import (
	"net/http"

	"go.uber.org/zap"
)

func Recover(l *zap.Logger) func(http.Handler) http.Handler {
	if l == nil {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					l.Error("panic recovered",
						zap.Any("panic", v),
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("remote", r.RemoteAddr),
						zap.Stack("stack"),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
