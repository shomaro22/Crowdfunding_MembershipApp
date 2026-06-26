package usecase

import (
	"testing"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func TestVerifyPurchaseBuyer(t *testing.T) {
	var (
		buyer = uuid.New()
		p     = &entity.Purchase{BuyerID: buyer}
	)
	if err := verifyPurchaseBuyer(p, buyer); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	if verifyPurchaseBuyer(p, uuid.New()) != ErrForbidden {
		t.Fatal("expected forbidden for other buyer")
	}
	if verifyPurchaseBuyer(nil, buyer) != ErrForbidden {
		t.Fatal("expected forbidden for nil purchase")
	}
}
