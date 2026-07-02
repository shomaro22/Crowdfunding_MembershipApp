package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConnectAccountCanCharge(t *testing.T) {
	var (
		enabled  ConnectAccount
		disabled ConnectAccount
	)

	enabled = ConnectAccount{ChargesEnabled: true}
	disabled = ConnectAccount{ChargesEnabled: false}
	if !enabled.CanCharge() {
		t.Fatal("charges enabled should allow charge")
	}
	if disabled.CanCharge() {
		t.Fatal("charges disabled should block charge")
	}
}

func TestRefreshTokenIsRevoked(t *testing.T) {
	var (
		now    = time.Now().UTC()
		active RefreshToken
		revoked RefreshToken
	)

	active = RefreshToken{RevokedAt: nil}
	revokedAt := now
	revoked = RefreshToken{RevokedAt: &revokedAt, ID: uuid.New()}
	if active.IsRevoked() {
		t.Fatal("nil revoked_at should not be revoked")
	}
	if !revoked.IsRevoked() {
		t.Fatal("revoked token should be revoked")
	}
}
