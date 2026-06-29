package usecase

import (
	"errors"
	"testing"

	"crowdfund-app/repository"
)

func TestMapRepoNotFound(t *testing.T) {
	var (
		err error
	)

	err = mapRepoNotFound(repository.ErrNotFound)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	other := errors.New("db down")
	err = mapRepoNotFound(other)
	if err != other {
		t.Fatalf("expected original error, got %v", err)
	}
}

func TestMapRepoNotFoundAs(t *testing.T) {
	var err error

	err = mapRepoNotFoundAs(repository.ErrNotFound, ErrForbidden)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
