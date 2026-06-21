package auth

import (
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func TestIssueParse(t *testing.T) {
	secret := []byte("01234567890123456789012345678901")
	svc := NewTokenService(secret, time.Minute)
	uid := uuid.New()
	adminRole := entity.AdminRoleSuperAdmin
	token, err := svc.Issue(AuthClaims{
		UserID:        uid,
		Role:          entity.UserRoleAdmin,
		TwoFAVerified: true,
		AdminRole:     &adminRole,
		TokenVersion:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != uid {
		t.Fatalf("user id mismatch")
	}
	if claims.Role != entity.UserRoleAdmin {
		t.Fatalf("role mismatch")
	}
	if !claims.TwoFAVerified {
		t.Fatalf("expected 2fa verified")
	}
	if claims.AdminRole == nil || *claims.AdminRole != adminRole {
		t.Fatalf("admin role mismatch")
	}
	if claims.TokenVersion != 1 {
		t.Fatalf("token version mismatch")
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if ComparePassword(hash, "secret123") != nil {
		t.Fatal("password should match")
	}
	if ComparePassword(hash, "wrong") == nil {
		t.Fatal("password should not match")
	}
}

func TestOpaqueToken(t *testing.T) {
	plain, err := OpaqueHex()
	if err != nil || len(plain) != 64 {
		t.Fatalf("expected 64 hex chars, got %q", plain)
	}
	h1 := OpaqueHash(plain)
	h2 := OpaqueHash(plain)
	if h1 != h2 || h1 == plain {
		t.Fatal("hash mismatch or not hashed")
	}
}
