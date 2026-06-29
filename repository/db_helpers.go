package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotFound = errors.New("not found")

func mapRecordNotFound(err error) error {
	if err == gorm.ErrRecordNotFound {
		return ErrNotFound
	}
	return err
}

func firstOne[T any](db *gorm.DB, dest *T) error {
	return mapRecordNotFound(db.First(dest).Error)
}

func firstOneForUpdate[T any](db *gorm.DB, dest *T) error {
	return mapRecordNotFound(
		db.Clauses(clause.Locking{Strength: "UPDATE"}).First(dest).Error,
	)
}
