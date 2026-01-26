package password

import (
	"errors"
	"testing"
)

func TestHashPassword_And_CheckPassword_OK(t *testing.T) {
	t.Parallel()

	pass := "super-secret"
	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatalf("HashPassword() returned empty hash")
	}

	if err := CheckPassword(hash, pass); err != nil {
		t.Fatalf("CheckPassword() error = %v", err)
	}
}

func TestCheckPassword_WrongPassword_ReturnsErrWrongPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("right-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	err = CheckPassword(hash, "wrong-password")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("want ErrWrongPassword, got %v", err)
	}
}

func TestHashPassword_SamePasswordDifferentHashes(t *testing.T) {
	t.Parallel()

	pass := "same-password"

	h1, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	h2, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if h1 == h2 {
		t.Fatalf("expected different hashes for the same password, got equal")
	}
}
