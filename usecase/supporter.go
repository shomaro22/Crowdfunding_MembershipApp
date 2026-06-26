package usecase

import (
	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"gorm.io/gorm"
)

type SupporterView struct {
	Supporter   *entity.ProjectSupporter
	Project     *entity.Project
	Plan        *entity.ProjectPlan
	PendingPlan *entity.ProjectPlan
}

type UpgradeOut struct {
	Supporter               SupporterView
	PaymentIntentClientSecret *string
}

type SupporterUseCase struct {
	DB         *gorm.DB
	Users      repository.UserRepo
	Projects   repository.ProjectRepo
	Supporters repository.SupporterRepo
	Purchases  repository.PurchaseRepo
	Connect    repository.ConnectRepo
	Stripe     stripe.Client
}

type PurchaseView struct {
	Purchase *entity.Purchase
	Project  *entity.Project
	Buyer    *entity.User
}

type PurchaseUseCase struct {
	Users     repository.UserRepo
	Projects  repository.ProjectRepo
	Purchases repository.PurchaseRepo
}
