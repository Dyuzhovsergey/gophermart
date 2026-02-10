package withdrawals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/withdrawalsrepo"
)

type fakeRepo struct {
	withdrawCalled bool
	gotUserID      int64
	gotOrder       string
	gotSum         float64

	withdrawErr error

	listCalled bool
	listRes    []withdrawalsrepo.Withdrawal
	listErr    error
}

func (f *fakeRepo) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	f.withdrawCalled = true
	f.gotUserID = userID
	f.gotOrder = order
	f.gotSum = sum
	return f.withdrawErr
}

func (f *fakeRepo) ListWithdrawals(ctx context.Context, userID int64) ([]withdrawalsrepo.Withdrawal, error) {
	f.listCalled = true
	f.gotUserID = userID
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listRes, nil
}

func TestService_Withdraw_InvalidOrder_ReturnsErrInvalidOrder_AndDoesNotCallRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo)

	err := svc.Withdraw(context.Background(), 1, "abc", 10)

	if err != domainerr.ErrInvalidOrder {
		t.Fatalf("want %v, got %v", domainerr.ErrInvalidOrder, err)
	}
	if repo.withdrawCalled {
		t.Fatalf("repo.Withdraw should NOT be called on invalid order")
	}
}

func TestService_Withdraw_SumNotPositive_ReturnsErrConflict_AndDoesNotCallRepo(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo)

	err := svc.Withdraw(context.Background(), 1, "79927398713", 0)

	if err != domainerr.ErrConflict {
		t.Fatalf("want %v, got %v", domainerr.ErrConflict, err)
	}
	if repo.withdrawCalled {
		t.Fatalf("repo.Withdraw should NOT be called on non-positive sum")
	}
}

func TestService_Withdraw_OK_CallsRepoWithSameArgs(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo)

	userID := int64(42)
	order := "79927398713"
	sum := 123.45

	err := svc.Withdraw(context.Background(), userID, order, sum)
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}

	if !repo.withdrawCalled {
		t.Fatalf("repo.Withdraw should be called")
	}
	if repo.gotUserID != userID {
		t.Fatalf("want userID=%d, got %d", userID, repo.gotUserID)
	}
	if repo.gotOrder != order {
		t.Fatalf("want order=%q, got %q", order, repo.gotOrder)
	}
	if repo.gotSum != sum {
		t.Fatalf("want sum=%v, got %v", sum, repo.gotSum)
	}
}

func TestService_Withdraw_RepoError_IsReturned(t *testing.T) {
	wantErr := errors.New("db error")

	repo := &fakeRepo{withdrawErr: wantErr}
	svc := New(repo)

	err := svc.Withdraw(context.Background(), 1, "79927398713", 10)
	if err != wantErr {
		t.Fatalf("want %v, got %v", wantErr, err)
	}
	if !repo.withdrawCalled {
		t.Fatalf("repo.Withdraw should be called")
	}
}

func TestService_List_OK_ProxyToRepo(t *testing.T) {
	ts := time.Date(2026, 1, 19, 12, 0, 0, 0, time.UTC)
	want := []withdrawalsrepo.Withdrawal{
		{Order: "123", Sum: 10, ProcessedAt: ts},
	}

	repo := &fakeRepo{listRes: want}
	svc := New(repo)

	got, err := svc.List(context.Background(), 7)
	if err != nil {
		t.Fatalf("want nil err, got %v", err)
	}
	if !repo.listCalled {
		t.Fatalf("repo.ListWithdrawals should be called")
	}
	if repo.gotUserID != 7 {
		t.Fatalf("want userID=7, got %d", repo.gotUserID)
	}

	if len(got) != 1 {
		t.Fatalf("want 1 item, got %d", len(got))
	}
	if got[0].Order != "123" || got[0].Sum != 10 || !got[0].ProcessedAt.Equal(ts) {
		t.Fatalf("unexpected item: %+v", got[0])
	}
}

func TestService_List_RepoError_IsReturned(t *testing.T) {
	wantErr := errors.New("boom")

	repo := &fakeRepo{listErr: wantErr}
	svc := New(repo)

	_, err := svc.List(context.Background(), 7)
	if err != wantErr {
		t.Fatalf("want %v, got %v", wantErr, err)
	}
	if !repo.listCalled {
		t.Fatalf("repo.ListWithdrawals should be called")
	}
}
