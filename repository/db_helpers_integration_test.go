//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestFirstOneReturnsErrNotFound(t *testing.T) {
	var (
		gdb = openIntegrationDB(t)
		row entity.Purchase
		err error
	)

	err = repository.FirstOneForTest(gdb.Where("id = ?", uuid.New()), &row)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFirstOneForUpdateReturnsErrNotFound(t *testing.T) {
	var (
		gdb = openIntegrationDB(t)
		row entity.Purchase
		err error
	)

	err = repository.FirstOneForUpdateForTest(gdb.Where("id = ?", uuid.New()), &row)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetOneReturnsErrNotFound(t *testing.T) {
	var (
		gdb = openIntegrationDB(t)
		err error
	)

	_, err = repository.GetOneForTest[entity.Purchase](gdb.Where("id = ?", uuid.New()))
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPurchaseFindIDReturnsErrNotFound(t *testing.T) {
	var (
		gdb  = openIntegrationDB(t)
		repo = repository.NewPurchaseRepo(gdb)
		ctx  = context.Background()
		row  *entity.Purchase
		err  error
	)

	row, err = repo.FindID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for missing purchase")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if row != nil {
		t.Fatalf("expected nil purchase, got %+v", row)
	}
}

func TestPurchaseFindIDReturnsRow(t *testing.T) {
	var (
		gdb      = openIntegrationDB(t)
		repo     = repository.NewPurchaseRepo(gdb)
		ctx      = context.Background()
		buyer    = seedIntegrationUser(t, gdb, entity.UserRoleBuyer, nil)
		seller   = seedIntegrationUser(t, gdb, entity.UserRoleSeller, nil)
		project  = seedIntegrationProject(t, gdb, seller.ID)
		purchase = seedIntegrationPurchase(t, gdb, buyer.ID, project.ID)
		row      *entity.Purchase
		err      error
	)

	row, err = repo.FindID(ctx, purchase.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil || row.ID != purchase.ID {
		t.Fatalf("unexpected purchase: %+v", row)
	}
}

func seedIntegrationProject(t *testing.T, gdb *gorm.DB, sellerID uuid.UUID) *entity.Project {
	var (
		now time.Time
		p   *entity.Project
		err error
	)
	t.Helper()
	now = time.Now().UTC()
	p = &entity.Project{
		ID:             uuid.New(),
		SellerID:       sellerID,
		Title:          fmt.Sprintf("project-%s", uuid.New()),
		Description:    "integration test project",
		TargetAmount:   10000,
		DurationType:   entity.DurationUnlimited,
		StartDate:      now,
		Status:         entity.ProjectStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err = gdb.Create(p).Error
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&entity.Project{}, "id = ?", p.ID).Error
	})
	return p
}

func seedIntegrationPurchase(t *testing.T, gdb *gorm.DB, buyerID, projectID uuid.UUID) *entity.Purchase {
	var (
		now time.Time
		p   *entity.Purchase
		err error
	)
	t.Helper()
	now = time.Now().UTC()
	p = &entity.Purchase{
		ID:            uuid.New(),
		ProjectID:     projectID,
		BuyerID:       buyerID,
		Quantity:      1,
		TotalPrice:    500,
		Status:        entity.PurchaseStatusPending,
		PaymentMethod: entity.PaymentMethodCard,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = gdb.Create(p).Error
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&entity.Purchase{}, "id = ?", p.ID).Error
	})
	return p
}
