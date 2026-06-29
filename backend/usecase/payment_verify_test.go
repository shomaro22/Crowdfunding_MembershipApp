package usecase

import (
	"testing"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

func TestVerifyPurchaseBuyer(t *testing.T) {
	var (
		buyer = uuid.New()
		p     = model.Purchase{BuyerID: buyer}
	)
	if err := verifyPurchaseBuyer(p, buyer); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	if verifyPurchaseBuyer(p, uuid.New()) != ErrForbidden {
		t.Fatal("expected forbidden for other buyer")
	}
	if verifyPurchaseBuyer(model.Purchase{}, buyer) != ErrForbidden {
		t.Fatal("expected forbidden for zero purchase")
	}
}
