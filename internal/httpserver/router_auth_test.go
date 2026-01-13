package httpserver_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver"
	"github.com/Dyuzhovsergey/gophermart/internal/service/domainerr"
)

func TestPOST_Register_OK_Returns200AndTokenJSON(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{registerToken: "token-123"},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/register",
		bytes.NewBufferString(`{"login":"sergey","password":"qwerty"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["token"] != "token-123" {
		t.Fatalf("want token=%q, got %v", "token-123", resp["token"])
	}
}

func TestPOST_Register_BadJSON_Returns400(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{registerToken: "x"},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/register",
		bytes.NewBufferString(`{"login":`)) // битый JSON
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPOST_Register_EmptyFields_Returns400(t *testing.T) {
	tests := []string{
		`{"login":"","password":"p"}`,
		`{"login":"u","password":""}`,
	}

	for _, body := range tests {
		r := httpserver.NewRouter(httpserver.Deps{
			Logger: nil,
			Auth:   &fakeAuth{registerToken: "x"},
			JWT:    &fakeJWT{userID: 1},
			Orders: &fakeOrdersService{},
		})

		req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("body=%s want %d, got %d", body, http.StatusBadRequest, w.Code)
		}
	}
}

func TestPOST_Register_Conflict_Returns409(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{registerErr: domainerr.ErrConflict},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/register",
		bytes.NewBufferString(`{"login":"sergey","password":"qwerty"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("want %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestPOST_Register_InternalError_Returns500(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{registerErr: errors.New("db down")},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/register",
		bytes.NewBufferString(`{"login":"sergey","password":"qwerty"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestPOST_Login_OK_Returns200AndTokenJSON(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{loginToken: "token-xyz"},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	req.SetBasicAuth("sergey", "qwerty")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["token"] != "token-xyz" {
		t.Fatalf("want token=%q, got %v", "token-xyz", resp["token"])
	}
}

func TestPOST_Login_NoBasicAuth_Returns400(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{loginToken: "x"},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPOST_Login_Unauthorized_Returns401(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{loginErr: domainerr.ErrUnauthorized},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	req.SetBasicAuth("sergey", "wrong")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestPOST_Login_InternalError_Returns500(t *testing.T) {
	r := httpserver.NewRouter(httpserver.Deps{
		Logger: nil,
		Auth:   &fakeAuth{loginErr: errors.New("boom")},
		JWT:    &fakeJWT{userID: 1},
		Orders: &fakeOrdersService{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	req.SetBasicAuth("sergey", "qwerty")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
