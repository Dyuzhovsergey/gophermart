package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyuzhovsergey/gophermart/internal/httpserver/userctx"
)

type fakeVerifier struct {
	wantToken string
	userID    int64
	err       error
	called    bool
}

func (f *fakeVerifier) Verify(tokenString string) (int64, error) {
	f.called = true
	if f.wantToken != "" && tokenString != f.wantToken {
		return 0, errors.New("bad token")
	}
	if f.err != nil {
		return 0, f.err
	}
	return f.userID, nil
}

func TestBearerAuth_NoAuthorizationHeader_Returns401(t *testing.T) {
	v := &fakeVerifier{wantToken: "ok", userID: 1}
	mw := BearerAuth(v)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	w := httptest.NewRecorder()

	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if nextCalled {
		t.Fatalf("next handler should NOT be called")
	}
}

func TestBearerAuth_BadScheme_Returns401(t *testing.T) {
	v := &fakeVerifier{wantToken: "ok", userID: 1}
	mw := BearerAuth(v)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next handler should NOT be called")
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	r.Header.Set("Authorization", "Basic abc")
	w := httptest.NewRecorder()

	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestBearerAuth_VerifierError_Returns401(t *testing.T) {
	v := &fakeVerifier{wantToken: "ok", err: errors.New("verify failed")}
	mw := BearerAuth(v)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next handler should NOT be called")
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	r.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()

	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want %d, got %d", http.StatusUnauthorized, w.Code)
	}
	if !v.called {
		t.Fatalf("expected verifier.Verify to be called")
	}
}

func TestBearerAuth_OK_PutsUserIDToContextAndCallsNext(t *testing.T) {
	v := &fakeVerifier{wantToken: "ok", userID: 42}
	mw := BearerAuth(v)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true

		id, ok := userctx.UserID(r.Context())
		if !ok {
			t.Fatalf("expected userID in context")
		}
		if id != 42 {
			t.Fatalf("want userID=42, got %d", id)
		}

		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	r.Header.Set("Authorization", "Bearer ok")
	w := httptest.NewRecorder()

	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want %d, got %d", http.StatusOK, w.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next handler to be called")
	}
}
