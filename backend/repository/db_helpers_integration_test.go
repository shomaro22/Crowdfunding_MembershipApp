//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestFirstOneReturnsErrNotFound(t *testing.T) {
	var (
		gdb = openIntegrationDB(t)
		row model.Purchase
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
		row model.Purchase
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

	_, err = repository.GetOneForTest[model.Purchase](gdb.Where("id = ?", uuid.New()))
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPurchaseGetIDReturnsErrNotFound(t *testing.T) {
	var (
		gdb  = openIntegrationDB(t)
		repo = repository.NewPurchaseRepo(gdb)
		ctx  = context.Background()
		row  model.Purchase
		err  error
	)

	row, err = repo.GetID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for missing purchase")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if row.ID != uuid.Nil {
		t.Fatalf("expected nil purchase, got %+v", row)
	}
}

func TestPurchaseGetIDReturnsRow(t *testing.T) {
	var (
		gdb      = openIntegrationDB(t)
		repo     = repository.NewPurchaseRepo(gdb)
		ctx      = context.Background()
		buyer    = seedIntegrationUser(t, gdb, model.UserRoleBuyer, nil)
		seller   = seedIntegrationUser(t, gdb, model.UserRoleSeller, nil)
		project  = seedIntegrationProject(t, gdb, seller.ID)
		purchase = seedIntegrationPurchase(t, gdb, buyer.ID, project.ID)
		row      model.Purchase
		err      error
	)

	row, err = repo.GetID(ctx, purchase.ID)
	if err != nil {
		t.Fatal(err)
	}
	if row.ID != purchase.ID {
		t.Fatalf("unexpected purchase: %+v", row)
	}
}

func seedIntegrationProject(t *testing.T, gdb *gorm.DB, sellerID uuid.UUID) *model.Project {
	var (
		now time.Time
		p   *model.Project
		err error
	)
	t.Helper()
	now = time.Now().UTC()
	p = &model.Project{
		ID:             uuid.New(),
		SellerID:       sellerID,
		Title:          fmt.Sprintf("project-%s", uuid.New()),
		Description:    "integration test project",
		TargetAmount:   10000,
		DurationType:   model.DurationUnlimited,
		StartDate:      now,
		Status:         model.ProjectStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err = gdb.Create(p).Error
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&model.Project{}, "id = ?", p.ID).Error
	})
	return p
}

func seedIntegrationPurchase(t *testing.T, gdb *gorm.DB, buyerID, projectID uuid.UUID) *model.Purchase {
	var (
		now time.Time
		p   *model.Purchase
		err error
	)
	t.Helper()
	now = time.Now().UTC()
	p = &model.Purchase{
		ID:            uuid.New(),
		ProjectID:     projectID,
		BuyerID:       buyerID,
		Quantity:      1,
		TotalPrice:    500,
		Status:        model.PurchaseStatusPending,
		PaymentMethod: model.PaymentMethodCard,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = gdb.Create(p).Error
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = gdb.Delete(&model.Purchase{}, "id = ?", p.ID).Error
	})
	return p
}
