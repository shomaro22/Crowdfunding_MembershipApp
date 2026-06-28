package validator

import (
	"strings"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type AdminUserListInput struct {
	Role   string
	Search string
	Limit  int
	Offset int
}

type AdminUserListNormalized struct {
	Role   *entity.UserRole
	Search string
	Limit  int
	Offset int
}

type AdminUpdateUserInput struct {
	Email    *string
	Name     *string
	Role     *string
	IsActive *bool
}

type AdminUpdateUserNormalized struct {
	Email       *string
	Name        *string
	NameSet     bool
	Role        *entity.UserRole
	RoleSet     bool
	IsActive    *bool
	IsActiveSet bool
}

type AdminPurchaseListInput struct {
	Status string
	Limit  int
	Offset int
}

type AdminPurchaseListNormalized struct {
	Status *entity.PurchaseStatus
	Limit  int
	Offset int
}

func ValidateAdminUserList(in AdminUserListInput) (AdminUserListNormalized, error) {
	var (
		out  AdminUserListNormalized
		role entity.UserRole
		err  error
	)
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return AdminUserListNormalized{}, err
	}
	out.Search = strings.TrimSpace(in.Search)
	if in.Role != "" {
		role = entity.UserRole(strings.TrimSpace(strings.ToUpper(in.Role)))
		if !role.Valid() {
			return AdminUserListNormalized{}, entity.ErrBadRole
		}
		out.Role = &role
	}
	return out, nil
}

func ValidateAdminUpdateUser(in AdminUpdateUserInput) (AdminUpdateUserNormalized, error) {
	var (
		out  AdminUpdateUserNormalized
		role entity.UserRole
	)
	if in.Email == nil && in.Name == nil && in.Role == nil && in.IsActive == nil {
		return AdminUpdateUserNormalized{}, entity.ErrBadAccount
	}
	if in.Email != nil {
		normalized, err := entity.ValidateEmail(*in.Email)
		if err != nil {
			return AdminUpdateUserNormalized{}, err
		}
		out.Email = &normalized
	}
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			out.Name = nil
		} else {
			out.Name = &trimmed
		}
		out.NameSet = true
	}
	if in.Role != nil {
		role = entity.UserRole(strings.TrimSpace(strings.ToUpper(*in.Role)))
		if !role.Valid() {
			return AdminUpdateUserNormalized{}, entity.ErrBadRole
		}
		out.Role = &role
		out.RoleSet = true
	}
	if in.IsActive != nil {
		out.IsActive = in.IsActive
		out.IsActiveSet = true
	}
	return out, nil
}

func ValidateAdminPurchaseList(in AdminPurchaseListInput) (AdminPurchaseListNormalized, error) {
	var (
		out AdminPurchaseListNormalized
		st  entity.PurchaseStatus
		err error
	)
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return AdminPurchaseListNormalized{}, err
	}
	if strings.TrimSpace(in.Status) != "" {
		st = entity.PurchaseStatus(strings.TrimSpace(strings.ToUpper(in.Status)))
		if !st.Valid() {
			return AdminPurchaseListNormalized{}, entity.ErrBadPurchase
		}
		out.Status = &st
	}
	return out, nil
}

func ValidateAdminPurchaseStatus(status string) (entity.PurchaseStatus, error) {
	var (
		st  entity.PurchaseStatus
		err error
	)
	err = validation.Validate(strings.TrimSpace(status), validation.Required)
	if err != nil {
		return "", err
	}
	st = entity.PurchaseStatus(strings.TrimSpace(strings.ToUpper(status)))
	switch st {
	case entity.PurchaseStatusPaid, entity.PurchaseStatusCancelled, entity.PurchaseStatusRefunded:
		return st, nil
	default:
		return "", entity.ErrBadPurchase
	}
}

type SellerSupporterListInput struct {
	ProjectID *uuid.UUID
	Limit     int
	Offset    int
}

type SellerSupporterListNormalized struct {
	ProjectID *uuid.UUID
	Limit     int
	Offset    int
}

func ValidateSellerSupporterList(in SellerSupporterListInput) (SellerSupporterListNormalized, error) {
	var (
		out SellerSupporterListNormalized
		err error
	)
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return SellerSupporterListNormalized{}, err
	}
	out.ProjectID = in.ProjectID
	return out, nil
}
