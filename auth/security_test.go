package auth

import (
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func TestTokenVersion(t *testing.T) {
	var (
		secret []byte
		svc    *TokenService
		uid    uuid.UUID
		token  string
		err    error
		claims AuthClaims
	)

	secret = []byte("01234567890123456789012345678901")
	svc = NewTokenService(secret, time.Minute)
	uid = uuid.New()
	token, err = svc.Issue(AuthClaims{
		UserID:       uid,
		Role:         entity.UserRoleBuyer,
		TokenVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err = svc.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TokenVersion != 1 {
		t.Fatalf("expected tv=1, got %d", claims.TokenVersion)
	}
}

func TestPending2FAToken(t *testing.T) {
	var (
		secret []byte
		uid    uuid.UUID
		now    time.Time
		tok    string
		got    uuid.UUID
		err    error
	)

	secret = []byte("01234567890123456789012345678901")
	uid = uuid.New()
	now = time.Now().UTC()
	tok = SignPending2FA(secret, uid, now)
	got, err = VerifyPending2FA(secret, tok, now)
	if err != nil || got != uid {
		t.Fatalf("pending 2fa verify failed: %v", err)
	}
	_, err = VerifyPending2FA(secret, tok, now.Add(6*time.Minute))
	if err == nil {
		t.Fatal("expected expired pending token")
	}
}

func TestSecretRoundTrip(t *testing.T) {
	var (
		key   []byte
		enc   string
		err   error
		plain string
	)

	key = []byte("01234567890123456789012345678901")
	enc, err = EncryptSecret(key, "BASE32SECRET")
	if err != nil {
		t.Fatal(err)
	}
	plain, err = DecryptSecret(key, enc)
	if err != nil || plain != "BASE32SECRET" {
		t.Fatalf("decrypt mismatch: %q err=%v", plain, err)
	}
}
