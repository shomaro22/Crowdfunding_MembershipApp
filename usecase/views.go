package usecase

import (
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

type UserView struct {
	ID            uuid.UUID
	Email         string
	Name          *string
	Role          entity.UserRole
	AccountStatus entity.AccountStatus
	SellerStatus  entity.SellerStatus
	AdminRole     *entity.AdminRole
	TOTPEnabled   bool
	AvatarURL     *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (v UserView) IsActive() bool {
	return v.AccountStatus == entity.AccountStatusActive
}

func UserViewFrom(u *entity.User) UserView {
	if u == nil {
		return UserView{}
	}
	return UserView{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Role:          u.Role,
		AccountStatus: u.AccountStatus,
		SellerStatus:  u.SellerStatus,
		AdminRole:     u.AdminRole,
		TOTPEnabled:   u.TOTPEnabled,
		AvatarURL:     u.AvatarURL,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func UserViewPtr(u *entity.User) *UserView {
	if u == nil {
		return nil
	}
	v := UserViewFrom(u)
	return &v
}

type ConnectAccountView struct {
	StripeAccountID  string
	ChargesEnabled   bool
	PayoutsEnabled   bool
	DetailsSubmitted bool
}

func ConnectAccountViewFrom(row *entity.ConnectAccount) *ConnectAccountView {
	if row == nil {
		return nil
	}
	return &ConnectAccountView{
		StripeAccountID:  row.StripeAccountID,
		ChargesEnabled:   row.ChargesEnabled,
		PayoutsEnabled:   row.PayoutsEnabled,
		DetailsSubmitted: row.DetailsSubmitted,
	}
}

type SellerRegItemView struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	FullName     string
	BirthDate    time.Time
	Address      string
	PhotoPresent bool
	HasIDFront   bool
	HasIDBack    bool
	Status       entity.RegStatus
	RejectReason *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func SellerRegItemFrom(r *entity.SellerReg) SellerRegItemView {
	if r == nil {
		return SellerRegItemView{}
	}
	return SellerRegItemView{
		ID:           r.ID,
		UserID:       r.UserID,
		FullName:     r.FullName,
		BirthDate:    r.BirthDate,
		Address:      r.Address,
		PhotoPresent: r.PhotoPresent,
		HasIDFront:   r.HasIDFront(),
		HasIDBack:    r.HasIDBack(),
		Status:       r.Status,
		RejectReason: r.RejectReason,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}
