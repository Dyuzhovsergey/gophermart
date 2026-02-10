package jwt

import (
	"testing"
	"time"
)

func TestManager_GenerateAndVerify_OK(t *testing.T) {
	t.Parallel()

	m, err := New("secret", 24*time.Hour)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// фиксируем время для воспроизводимости
	fixed := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return fixed }

	token, err := m.Generate(42)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	got, err := m.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if got != 42 {
		t.Fatalf("Verify() got userID=%d, want %d", got, 42)
	}
}

func TestManager_Verify_Expired(t *testing.T) {
	t.Parallel()

	secret := "secret"
	ttl := 1 * time.Hour

	gen, err := New(secret, ttl)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	issuedAt := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	gen.now = func() time.Time { return issuedAt }

	token, err := gen.Generate(7)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	ver, err := New(secret, ttl)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// перематываем время дальше exp
	ver.now = func() time.Time { return issuedAt.Add(2 * time.Hour) }

	_, err = ver.Verify(token)
	if err == nil {
		t.Fatalf("Verify() expected error, got nil")
	}
	if err != ErrExpiredToken {
		t.Fatalf("Verify() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestManager_Verify_BadSignature(t *testing.T) {
	t.Parallel()

	gen, err := New("secret-1", 24*time.Hour)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	token, err := gen.Generate(100)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	ver, err := New("secret-2", 24*time.Hour)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = ver.Verify(token)
	if err == nil {
		t.Fatalf("Verify() expected error, got nil")
	}
	if err != ErrInvalidToken {
		t.Fatalf("Verify() error = %v, want %v", err, ErrInvalidToken)
	}
}
