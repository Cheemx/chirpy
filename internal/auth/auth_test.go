package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCheckPasswordHash(t *testing.T) {
	// First, we need to create some hashed passwords for testing
	password1 := "correctPassword123!"
	password2 := "anotherPassword456!"
	hash1, _ := HashPassword(password1)
	hash2, _ := HashPassword(password2)

	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{
			name:     "Correct password",
			password: password1,
			hash:     hash1,
			wantErr:  false,
		},
		{
			name:     "Incorrect password",
			password: "wrongPassword",
			hash:     hash1,
			wantErr:  true,
		},
		{
			name:     "Password doesn't match different hash",
			password: password1,
			hash:     hash2,
			wantErr:  true,
		},
		{
			name:     "Empty password",
			password: "",
			hash:     hash1,
			wantErr:  true,
		},
		{
			name:     "Invalid hash",
			password: password1,
			hash:     "invalidhash",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPasswordHash(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWT(t *testing.T) {
	const secret = "Cheems"

	t.Run("round trip ok", func(t *testing.T) {
		id := uuid.New()
		tok := mustMakeJWT(t, id, secret, 2*time.Second)

		gotID, err := ValidateJWT(tok, secret)
		if err != nil {
			t.Fatalf("ValidateJWT error: %v", err)
		}
		if gotID != id {
			t.Fatalf("got %s want %s", gotID, id)
		}
	})

	t.Run("wrong secret rejected", func(t *testing.T) {
		id := uuid.New()
		tok := mustMakeJWT(t, id, secret, 2*time.Second)

		if _, err := ValidateJWT(tok, "cheems"); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("expired token rejected", func(t *testing.T) {
		id := uuid.New()
		tok := mustMakeJWT(t, id, secret, -1*time.Second)

		if _, err := ValidateJWT(tok, secret); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func mustMakeJWT(t *testing.T, id uuid.UUID, secret string, d time.Duration) string {
	t.Helper()
	tok, err := MakeJWT(id, secret, d)
	if err != nil {
		t.Fatalf("MakeJWT error: %v", err)
	}
	return tok
}
