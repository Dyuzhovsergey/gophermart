// Package httperrors contains the conversion of domain errors into HTTP codes
package httperrors

import (
	"net/http"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
)

// MapErrorToStatus преобразует доменную ошибку в HTTP-код.
func MapErrorToStatus(err error) int {
	switch err {
	case domainerr.ErrUnauthorized:
		return http.StatusUnauthorized

	case domainerr.ErrConflict, domainerr.ErrAlreadyUploadedByAnother:
		return http.StatusConflict

	case domainerr.ErrInvalidOrder:
		return http.StatusUnprocessableEntity

	case domainerr.ErrNoFunds:
		return http.StatusPaymentRequired

	case domainerr.ErrOrderNotFound:
		return http.StatusNotFound

	case domainerr.ErrAlreadyUploadedByUser:
		return http.StatusOK

	default:
		return http.StatusInternalServerError
	}
}
