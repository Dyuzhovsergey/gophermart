package accounts

import (
	"context"
	"errors"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/service/accountsrepo"
)

type fakeRepo struct {
	provideCalled bool
	provideUserID int64
	provideErr    error

	getCalled   bool
	getUserID   int64
	getCurrent  float64
	getWithdraw float64
	getErr      error
}

func (f *fakeRepo) ProvideAccount(ctx context.Context, userID int64) error {
	f.provideCalled = true
	f.provideUserID = userID
	return f.provideErr
}

func (f *fakeRepo) GetBalance(ctx context.Context, userID int64) (float64, float64, error) {
	f.getCalled = true
	f.getUserID = userID
	return f.getCurrent, f.getWithdraw, f.getErr
}

var _ accountsrepo.Repository = (*fakeRepo)(nil)

func TestService_ProvideAccount_CallsRepo(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc := New(repo)

	err := svc.ProvideAccount(context.Background(), 10)
	if err != nil {
		t.Fatalf("ProvideAccount() error = %v", err)
	}
	if !repo.provideCalled {
		t.Fatalf("expected repo.ProvideAccount to be called")
	}
	if repo.provideUserID != 10 {
		t.Fatalf("expected userID=10, got %d", repo.provideUserID)
	}
}

func TestService_GetBalance_OK(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{
		getCurrent:  123.45,
		getWithdraw: 7,
	}
	svc := New(repo)

	bal, err := svc.GetBalance(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}
	if !repo.getCalled {
		t.Fatalf("expected repo.GetBalance to be called")
	}
	if repo.getUserID != 42 {
		t.Fatalf("expected userID=42, got %d", repo.getUserID)
	}
	if bal.Current != 123.45 {
		t.Fatalf("Current = %v, want %v", bal.Current, 123.45)
	}
	if bal.Withdrawn != 7 {
		t.Fatalf("Withdrawn = %v, want %v", bal.Withdrawn, 7.0)
	}
}

func TestService_GetBalance_RepoError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{getErr: errors.New("db error")}
	svc := New(repo)

	_, err := svc.GetBalance(context.Background(), 1)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
