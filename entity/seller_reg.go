package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// SellerReg は出品者 KYC 申請（5.1.8）。運転免許証のみ。画像は storage key のみ保持する。
type SellerReg struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	FullName     string     `gorm:"column:full_name;not null"`
	BirthDate    time.Time  `gorm:"column:birth_date;type:date;not null"`
	Address      string     `gorm:"type:text;not null"`
	PhotoPresent bool       `gorm:"column:photo_present;not null"`
	IDFrontKey   *string    `gorm:"column:identity_document_front_key;type:varchar"`
	IDBackKey    *string    `gorm:"column:identity_document_back_key;type:varchar"`
	Status       RegStatus  `gorm:"not null;default:PENDING"`
	RejectReason *string    `gorm:"column:rejection_reason;type:text"`
	ReviewedAt   *time.Time `gorm:"column:reviewed_at"`
	ReviewedBy   *uuid.UUID `gorm:"type:uuid;column:reviewed_by"`
	CreatedAt    time.Time  `gorm:"not null"`
	UpdatedAt    time.Time  `gorm:"not null"`
}

func (SellerReg) TableName() string { return "seller_registrations" }

// IDDocType は KYC スコープ上常に運転免許証（DB カラムは将来 migration で固定値化可）。
func (SellerReg) IDDocType() IDDocType {
	return IDDriversLicense
}

func (r SellerReg) HasIDFront() bool {
	return r.IDFrontKey != nil && strings.TrimSpace(*r.IDFrontKey) != ""
}

func (r SellerReg) HasIDBack() bool {
	return r.IDBackKey != nil && strings.TrimSpace(*r.IDBackKey) != ""
}

func (r SellerReg) HasMetadata() bool {
	return ValidateSellerKYC(r.FullName, r.BirthDate, r.Address, r.PhotoPresent) == nil
}

// CanApprove は 5.1.8 の承認条件（PENDING・表裏画像・必須メタデータ）。
func (r SellerReg) CanApprove() bool {
	if r.Status != RegPending {
		return false
	}
	if !r.HasIDFront() || !r.HasIDBack() {
		return false
	}
	return r.HasMetadata()
}
