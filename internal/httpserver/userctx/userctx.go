// Package userctx содержит функции для работы с userID в context.
package userctx

import "context"

type userIDKey struct{}

// WithUserID кладёт userID в контекст запроса.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID возвращает userID из контекста.
func UserID(ctx context.Context) (int64, bool) {
	v := ctx.Value(userIDKey{})
	id, ok := v.(int64)
	return id, ok
}
