package repository

import "gorm.io/gorm"

func FirstOneForTest[T tableModel](db *gorm.DB, dest *T) error {
	return firstOne(db, dest)
}

func FirstOneForUpdateForTest[T tableModel](db *gorm.DB, dest *T) error {
	return firstOneForUpdate(db, dest)
}

func GetOneForTest[T tableModel](db *gorm.DB) (T, error) {
	return getOne[T](db)
}
