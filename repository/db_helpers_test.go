package repository

import (
	"errors"
	"testing"

	"crowdfund-app/entity"

	"gorm.io/gorm"
)

func TestMapRecordNotFound(t *testing.T) {
	var (
		err error
	)

	err = mapRecordNotFound(gorm.ErrRecordNotFound)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	err = mapRecordNotFound(nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	other := errors.New("db down")
	err = mapRecordNotFound(other)
	if err != other {
		t.Fatalf("expected original error, got %v", err)
	}
}

func TestErrNotFoundDistinct(t *testing.T) {
	if errors.Is(gorm.ErrRecordNotFound, ErrNotFound) {
		t.Fatal("ErrNotFound must not match gorm.ErrRecordNotFound directly")
	}
}

func TestTableModelConstraint(t *testing.T) {
	var _ tableModel = entity.Purchase{}
}
