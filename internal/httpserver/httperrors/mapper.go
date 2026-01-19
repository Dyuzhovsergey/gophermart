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
		return http.StatusUnauthorized // // 401

	case domainerr.ErrConflict, domainerr.ErrAlreadyUploadedByAnother:
		return http.StatusConflict // 409

	case domainerr.ErrInvalidOrder:
		return http.StatusUnprocessableEntity // 422

	case domainerr.ErrNoFunds:
		return http.StatusPaymentRequired // 402

	case domainerr.ErrOrderNotFound:
		return http.StatusNotFound // 404

	case domainerr.ErrAlreadyUploadedByUser:
		return http.StatusOK // 200

	default:
		return http.StatusInternalServerError // 500
	}
}
