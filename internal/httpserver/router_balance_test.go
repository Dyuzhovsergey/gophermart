package httpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver"
)

func TestGET_UserBalance_NoToken_Returns401(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger:   nil,
		Auth:     &fakeAuth{},
		JWT:      &fakeJWT{userID: 1},
		Orders:   &fakeOrdersService{},
		Accounts: &fakeAccountsService{current: 10, withdrawn: 2},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGET_UserBalance_OK_Returns200AndJSON(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger:   nil,
		Auth:     &fakeAuth{},
		JWT:      &fakeJWT{userID: 1},
		Orders:   &fakeOrdersService{},
		Accounts: &fakeAccountsService{current: 500.5, withdrawn: 42},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
	req.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}

	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if got["current"] != 500.5 {
		t.Fatalf("want current=500.5, got %v", got["current"])
	}
	// В JSON число 42 станет float64(42)
	if got["withdrawn"] != 42.0 {
		t.Fatalf("want withdrawn=42, got %v", got["withdrawn"])
	}
}
