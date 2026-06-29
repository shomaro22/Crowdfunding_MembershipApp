package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrNotFound はリポジトリ層の「行が存在しない」エラー。
// gorm.ErrRecordNotFound は mapRecordNotFound / firstOne 系でここに変換する。
// usecase 層では errors.Is(err, repository.ErrNotFound) を判定し、
// 必要に応じて usecase.ErrNotFound 等の業務エラーへマッピングする（F3）。
var ErrNotFound = errors.New("not found")

type tableModel interface {
	TableName() string
}

func mapRecordNotFound(err error) error {
	if err == gorm.ErrRecordNotFound {
		return ErrNotFound
	}
	return err
}

func firstOne[T tableModel](db *gorm.DB, dest *T) error {
	return mapRecordNotFound(db.First(dest).Error)
}

func firstOneForUpdate[T tableModel](db *gorm.DB, dest *T) error {
	return mapRecordNotFound(
		db.Clauses(clause.Locking{Strength: "UPDATE"}).First(dest).Error,
	)
}

func getOne[T tableModel](db *gorm.DB) (T, error) {
	var row T
	err := firstOne(db, &row)
	if err != nil {
		var zero T
		return zero, err
	}
	return row, nil
}

func getOneForUpdate[T tableModel](db *gorm.DB) (T, error) {
	var row T
	err := firstOneForUpdate(db, &row)
	if err != nil {
		var zero T
		return zero, err
	}
	return row, nil
}
