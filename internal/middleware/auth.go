// Package middleware содержит HTTP middleware.
package middleware

import (
	"net/http"
	"strings"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/userctx"
)

// TokenVerifier — минимальный интерфейс верификации токена.
type TokenVerifier interface {
	Verify(tokenString string) (int64, error)
}

// BearerAuth проверяет Authorization: Bearer <token> и кладёт userID в context.
func BearerAuth(verifier TokenVerifier) func(http.Handler) http.Handler {
	if verifier == nil {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unauthorized", http.StatusUnauthorized) // 401
			})
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if strings.TrimSpace(auth) == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized) // 401
				return
			}

			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)  // 401
				return
			}

			userID, err := verifier.Verify(parts[1])
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized) // 401
				return
			}

			ctx := userctx.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
