package httperrors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
)

func TestMapErrorToStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "unauthorized -> 401",
			err:  domainerr.ErrUnauthorized,
			want: http.StatusUnauthorized,
		},
		{
			name: "conflict -> 409",
			err:  domainerr.ErrConflict,
			want: http.StatusConflict,
		},
		{
			name: "already uploaded by another -> 409",
			err:  domainerr.ErrAlreadyUploadedByAnother,
			want: http.StatusConflict,
		},
		{
			name: "invalid order -> 422",
			err:  domainerr.ErrInvalidOrder,
			want: http.StatusUnprocessableEntity,
		},
		{
			name: "no funds -> 402",
			err:  domainerr.ErrNoFunds,
			want: http.StatusPaymentRequired,
		},
		{
			name: "order not found -> 404",
			err:  domainerr.ErrOrderNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "already uploaded by user -> 200",
			err:  domainerr.ErrAlreadyUploadedByUser,
			want: http.StatusOK,
		},
		{
			name: "unknown error -> 500",
			err:  errors.New("some unexpected error"),
			want: http.StatusInternalServerError,
		},
		{
			name: "nil error -> 500",
			err:  nil,
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MapErrorToStatus(tt.err)
			if got != tt.want {
				t.Fatalf("MapErrorToStatus(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
