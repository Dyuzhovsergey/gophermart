// internal/service/orders/service_test.go
package orders

import (
	"context"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
)

// фейковый репозиторий
type fakeRepo struct {
	err error
}

func (f fakeRepo) AddOrder(ctx context.Context, userID int64, number string) error {
	return f.err
}

func TestUploadOrder_AlreadyByAnother(t *testing.T) {
	svc := New(fakeRepo{err: domainerr.ErrAlreadyUploadedByAnother})

	err := svc.UploadOrder(context.Background(), 1, "79927398713")
	if err != domainerr.ErrAlreadyUploadedByAnother {
		t.Fatalf("want %v, got %v", domainerr.ErrAlreadyUploadedByAnother, err)
	}
}
