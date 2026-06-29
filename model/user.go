package model

import (
	"time"

	"github.com/google/uuid"
)

// User の account_status は DB 用。API の is_active は IsActive() で変換する。
type User struct {
	ID                 uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email              string        `gorm:"not null;uniqueIndex"`
	Name               *string       `gorm:"type:varchar"`
	PasswordHash       string        `gorm:"column:password_hash;not null"`
	AvatarURL          *string       `gorm:"column:avatar_url;type:varchar"`
	Role               UserRole      `gorm:"not null;default:BUYER"`
	AccountStatus      AccountStatus `gorm:"column:account_status;not null;default:PENDING"`
	SellerStatus       SellerStatus  `gorm:"column:seller_status;not null;default:NOT_APPLICABLE"`
	AdminRole          *AdminRole    `gorm:"column:admin_role;type:varchar"`
	TOTPEnabled        bool          `gorm:"column:totp_enabled;not null;default:false"`
	TOTPSecret         *string       `gorm:"column:totp_secret_encrypted;type:varchar"`
	TokenVersion       int64         `gorm:"column:token_version;not null;default:0"`
	CreatedAt          time.Time     `gorm:"not null"`
	UpdatedAt          time.Time     `gorm:"not null"`
}

func (User) TableName() string { return "users" }

func (u User) IsActive() bool {
	return u.AccountStatus == AccountStatusActive
}

func (u User) CanLogin() bool {
	if !u.AccountStatus.CanLogin() {
		return false
	}
	if u.Role == UserRoleSeller && u.SellerStatus != SellerStatusActive {
		return false
	}
	return true
}

func (u User) MustSetup2FA() bool {
	return u.Role == UserRoleAdmin || u.Role == UserRoleSeller
}

func (u User) NeedsTOTP() bool {
	return u.MustSetup2FA() && u.TOTPEnabled
}
