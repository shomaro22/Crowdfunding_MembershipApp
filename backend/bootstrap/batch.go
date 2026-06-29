package bootstrap

import (
	"crowdfund-app/usecase"

	"gorm.io/gorm"
)

type BatchApp struct {
	DB    *gorm.DB
	Batch *usecase.BatchUseCase
}

func OpenForBatch(opts *Options) (*BatchApp, error) {
	var (
		gormDB       *gorm.DB
		repos        Repositories
		stripeClient = stripeClientFrom(opts)
		err          error
	)
	gormDB, err = OpenDB()
	if err != nil {
		return nil, err
	}
	repos = NewRepositories(gormDB)
	return &BatchApp{
		DB:    gormDB,
		Batch: NewBatchUseCase(gormDB, repos, stripeClient),
	}, nil
}
