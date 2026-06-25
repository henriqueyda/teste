package auth

import (
	"testing"

	"github.com/henrique-yda/teste-tecnico-itau/internal/domain"
)

var testSecret = []byte("test-secret")

func TestPasswordHashVerify(t *testing.T) {
	h, err := HashPassword("demo123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := VerifyPassword("demo123", h)
	if err != nil || !ok {
		t.Fatalf("expected match, got ok=%v err=%v", ok, err)
	}
	if ok, _ := VerifyPassword("wrong", h); ok {
		t.Fatal("expected mismatch for wrong password")
	}
}

func TestUserTokenRoundTrip(t *testing.T) {
	want := domain.Subject{UserID: "usr_x", CustomerID: "cust_x", Roles: []string{"customer"}}

	tok, err := MintUserToken(testSecret, "iss", "aud", want)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	got, err := VerifyUserToken(testSecret, "iss", "aud", tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.UserID != want.UserID || got.CustomerID != want.CustomerID || len(got.Roles) != 1 || got.Roles[0] != "customer" {
		t.Fatalf("subject mismatch: %+v", got)
	}

	// Wrong audience must fail (audience scoping).
	if _, err := VerifyUserToken(testSecret, "iss", "other-aud", tok); err == nil {
		t.Fatal("expected failure on wrong audience")
	}
	// Tampered token must fail (signature check).
	if _, err := VerifyUserToken(testSecret, "iss", "aud", tok[:len(tok)-2]+"xx"); err == nil {
		t.Fatal("expected failure on tampered token")
	}
}

