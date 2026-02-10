package orders

import (
	"context"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/ordersrepo"
)

type fakeRepo struct {
	err    error
	called bool
}

func (f *fakeRepo) AddOrder(ctx context.Context, userID int64, number string) error {
	f.called = true
	return f.err
}

func (f *fakeRepo) ListOrdersByUser(ctx context.Context, userID int64) ([]ordersrepo.Order, error) {
	return nil, nil
}

func TestUploadOrder_OK_CallsRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo)

	// Валидный по Луну номер
	err := svc.UploadOrder(context.Background(), 1, "79927398713")
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if !repo.called {
		t.Fatalf("expected repo.AddOrder to be called")
	}
}

func TestUploadOrder_InvalidNumber_DoesNotCallRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo)

	// Не проходит Лун
	err := svc.UploadOrder(context.Background(), 1, "79927398710")
	if err != domainerr.ErrInvalidOrder {
		t.Fatalf("want %v, got %v", domainerr.ErrInvalidOrder, err)
	}
	if repo.called {
		t.Fatalf("repo.AddOrder should NOT be called on invalid number")
	}
}

func TestUploadOrder_RepoError_PassedThrough(t *testing.T) {
	repo := &fakeRepo{err: domainerr.ErrAlreadyUploadedByAnother}
	svc := New(repo)

	err := svc.UploadOrder(context.Background(), 1, "79927398713")
	if err != domainerr.ErrAlreadyUploadedByAnother {
		t.Fatalf("want %v, got %v", domainerr.ErrAlreadyUploadedByAnother, err)
	}
	if !repo.called {
		t.Fatalf("expected repo.AddOrder to be called")
	}
}
