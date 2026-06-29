package model

import "time"

func CanSupport(user User, project Project, now time.Time) error {
	if user.Role != UserRoleBuyer {
		return ErrNotActive
	}
	if !user.CanLogin() {
		return ErrNotActive
	}
	if !project.IsPublic(now) {
		return ErrNotPublic
	}
	return nil
}

func CanViewContent(supporter ProjectSupporter, plan MembershipType, content ProjectContent, now time.Time) error {
	if !supporter.HasBenefits(now) {
		return ErrNoBenefits
	}
	if !plan.MeetsMinimum(content.MinimumPlan) {
		return ErrPlanTooLow
	}
	return nil
}

func CanMessage(supporter ProjectSupporter, now time.Time) error {
	if !supporter.HasBenefits(now) {
		return ErrNoBenefits
	}
	return nil
}

func NewBuyer(email, passwordHash string, name *string) User {
	return User{
		Email:         email,
		Name:          name,
		PasswordHash:  passwordHash,
		Role:          UserRoleBuyer,
		AccountStatus: AccountStatusPending,
		SellerStatus:  SellerStatusNotApplicable,
	}
}

func NewSeller(email, passwordHash string) User {
	return User{
		Email:         email,
		PasswordHash:  passwordHash,
		Role:          UserRoleSeller,
		AccountStatus: AccountStatusPending,
		SellerStatus:  SellerStatusPending,
	}
}
