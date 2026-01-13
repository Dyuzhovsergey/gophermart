package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver"
	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
	"github.com/Dyuzhovsergey/gophermart/internal/service/ordersrepo"
)

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

type fakeAuth struct{}

func (f *fakeAuth) Register(ctx context.Context, login, plainPassword string) (string, error) {
	return "token", nil
}

func (f *fakeAuth) LoginBasic(ctx context.Context, login, plainPassword string) (string, error) {
	return "token", nil
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

// ---- тесты ----

func TestPOST_UserOrders_NoToken_Returns401(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestPOST_UserOrders_NewOrder_Returns202(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{uploadErr: nil},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("want %d, got %d", http.StatusAccepted, w.Code)
	}
}

func TestPOST_UserOrders_AlreadyBySameUser_Returns200(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{uploadErr: domainerr.ErrAlreadyUploadedByUser},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}
}

func TestPOST_UserOrders_AlreadyByAnother_Returns409(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{uploadErr: domainerr.ErrAlreadyUploadedByAnother},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewBufferString("79927398713"))
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("want %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestGET_UserOrders_Empty_Returns204(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{list: nil},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestGET_UserOrders_OK_Returns200AndJSON(t *testing.T) {
	ts := time.Date(2026, 1, 5, 12, 0, 0, 0, time.FixedZone("MSK", 3*3600))
	accrual := 500.0

	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{
			list: []ordersrepo.Order{
				{
					Number:     "9278923470",
					Status:     "PROCESSED",
					Accrual:    &accrual,
					UploadedAt: ts,
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}

	var got []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 item, got %d", len(got))
	}

	if got[0]["number"] != "9278923470" {
		t.Fatalf("want number=9278923470, got %v", got[0]["number"])
	}
	if got[0]["status"] != "PROCESSED" {
		t.Fatalf("want status=PROCESSED, got %v", got[0]["status"])
	}
	if got[0]["uploaded_at"] != ts.Format(time.RFC3339) {
		t.Fatalf("want uploaded_at=%s, got %v", ts.Format(time.RFC3339), got[0]["uploaded_at"])
	}
	if got[0]["accrual"] != accrual {
		t.Fatalf("want accrual=%v, got %v", accrual, got[0]["accrual"])
	}
}
