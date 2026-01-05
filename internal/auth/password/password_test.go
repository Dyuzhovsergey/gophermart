package password

import "testing"

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

	ok, err := CheckPassword(hash, pass)
	if err != nil {
		t.Fatalf("CheckPassword() error = %v", err)
	}
	if !ok {
		t.Fatalf("CheckPassword() expected ok=true")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	t.Parallel()

	hash, err := HashPassword("right-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	ok, err := CheckPassword(hash, "wrong-password")
	if err != nil {
		t.Fatalf("CheckPassword() error = %v", err)
	}
	if ok {
		t.Fatalf("CheckPassword() expected ok=false")
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
