package validator

import (
	"strings"

	"crowdfund-app/model"

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
	Role   *model.UserRole
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
	Role        *model.UserRole
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
	Status *model.PurchaseStatus
	Limit  int
	Offset int
}

func ValidateAdminUserList(in AdminUserListInput) (AdminUserListNormalized, error) {
	var (
		out  AdminUserListNormalized
		role model.UserRole
		err  error
	)
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return AdminUserListNormalized{}, err
	}
	out.Search = strings.TrimSpace(in.Search)
	if in.Role != "" {
		role = model.UserRole(strings.TrimSpace(strings.ToUpper(in.Role)))
		if !role.Valid() {
			return AdminUserListNormalized{}, model.ErrBadRole
		}
		out.Role = &role
	}
	return out, nil
}

func ValidateAdminUpdateUser(in AdminUpdateUserInput) (AdminUpdateUserNormalized, error) {
	var (
		out  AdminUpdateUserNormalized
		role model.UserRole
	)
	if in.Email == nil && in.Name == nil && in.Role == nil && in.IsActive == nil {
		return AdminUpdateUserNormalized{}, model.ErrBadAccount
	}
	if in.Email != nil {
		normalized, err := model.ValidateEmail(*in.Email)
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
		role = model.UserRole(strings.TrimSpace(strings.ToUpper(*in.Role)))
		if !role.Valid() {
			return AdminUpdateUserNormalized{}, model.ErrBadRole
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
		st  model.PurchaseStatus
		err error
	)
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return AdminPurchaseListNormalized{}, err
	}
	if strings.TrimSpace(in.Status) != "" {
		st = model.PurchaseStatus(strings.TrimSpace(strings.ToUpper(in.Status)))
		if !st.Valid() {
			return AdminPurchaseListNormalized{}, model.ErrBadPurchase
		}
		out.Status = &st
	}
	return out, nil
}

func ValidateAdminPurchaseStatus(status string) (model.PurchaseStatus, error) {
	var (
		st  model.PurchaseStatus
		err error
	)
	err = validation.Validate(strings.TrimSpace(status), validation.Required)
	if err != nil {
		return "", err
	}
	st = model.PurchaseStatus(strings.TrimSpace(strings.ToUpper(status)))
	switch st {
	case model.PurchaseStatusPaid, model.PurchaseStatusCancelled, model.PurchaseStatusRefunded:
		return st, nil
	default:
		return "", model.ErrBadPurchase
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
