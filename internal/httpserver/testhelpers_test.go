package httpserver_test

import (
	"context"

	"github.com/Dyuzhovsergey/gophermart/internal/models"
	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/ordersrepo"
)

// ---- общие фейки для DI router_*_test.go) ----

// domainerr
var _ = domainerr.ErrUnauthorized

type fakeJWT struct {
	userID int64
	err    error
}

func (f *fakeJWT) Verify(tokenString string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.userID, nil
}

type fakeOrdersService struct {
	uploadErr error
	list      []ordersrepo.Order
	listErr   error
}

func (f *fakeOrdersService) UploadOrder(ctx context.Context, userID int64, number string) error {
	return f.uploadErr
}

func (f *fakeOrdersService) ListOrders(ctx context.Context, userID int64) ([]ordersrepo.Order, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.list, nil
}

type fakeAuth struct {
	registerToken string
	registerErr   error

	loginToken string
	loginErr   error
}

func (f *fakeAuth) Register(ctx context.Context, login, plainPassword string) (string, error) {
	if f.registerToken == "" && f.registerErr == nil {
		return "token", nil
	}
	return f.registerToken, f.registerErr
}

func (f *fakeAuth) LoginBasic(ctx context.Context, login, plainPassword string) (string, error) {
	if f.loginToken == "" && f.loginErr == nil {
		return "token", nil
	}
	return f.loginToken, f.loginErr
}

type fakeAccountsService struct {
	current   float64
	withdrawn float64
	err       error
}

func (f *fakeAccountsService) ProvideAccount(ctx context.Context, userID int64) error {
	return nil
}

func (f *fakeAccountsService) GetBalance(ctx context.Context, userID int64) (models.Balance, error) {
	if f.err != nil {
		return models.Balance{}, f.err
	}
	return models.Balance{Current: f.current, Withdrawn: f.withdrawn}, nil
}
