package entity

import (
	"time"

	"github.com/google/uuid"
)

// 身分証は storage key のみ保持し、公開 URL は持たない。
type SellerReg struct {
	ID               uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID    `gorm:"type:uuid;not null;index"`
	LastName         string       `gorm:"column:last_name;not null"`
	FirstName        string       `gorm:"column:first_name;not null"`
	LastNameKana     string       `gorm:"column:last_name_kana;not null"`
	FirstNameKana    string       `gorm:"column:first_name_kana;not null"`
	BirthYear        int          `gorm:"column:birth_year;not null"`
	BirthMonth       int          `gorm:"column:birth_month;not null"`
	BirthDay         int          `gorm:"column:birth_day;not null"`
	Gender           string       `gorm:"not null"`
	PostalCode       string       `gorm:"column:postal_code;not null"`
	Prefecture       string       `gorm:"not null"`
	City             string       `gorm:"not null"`
	Building         *string      `gorm:"type:varchar"`
	Occupation       string       `gorm:"not null"`
	Purpose          string       `gorm:"not null"`
	Nationality      string       `gorm:"not null"`
	HasBusiness      bool         `gorm:"column:business_registration;not null"`
	Revenue          *int         `gorm:"column:revenue"`
	Capital          *int         `gorm:"column:capital"`
	IsForeign        bool         `gorm:"column:is_foreign_national;not null;default:false"`
	IDDocType        IDDocType    `gorm:"column:identity_document_type;not null"`
	IDFrontKey       *string      `gorm:"column:identity_document_front_key;type:varchar"`
	IDBackKey        *string      `gorm:"column:identity_document_back_key;type:varchar"`
	ForeignDocKeys   JSONMetadata `gorm:"column:foreign_document_keys;type:jsonb"`
	Status           RegStatus    `gorm:"not null;default:PENDING"`
	RejectReason     *string      `gorm:"column:rejection_reason;type:text"`
	Score            int          `gorm:"not null;default:0"`
	ReviewedAt       *time.Time   `gorm:"column:reviewed_at"`
	ReviewedBy       *uuid.UUID   `gorm:"type:uuid;column:reviewed_by"`
	CreatedAt        time.Time    `gorm:"not null"`
	UpdatedAt        time.Time    `gorm:"not null"`
}

func (SellerReg) TableName() string { return "seller_registrations" }

func (r SellerReg) HasIDFront() bool {
	return r.IDFrontKey != nil && *r.IDFrontKey != ""
}

func (r SellerReg) HasIDBack() bool {
	return r.IDBackKey != nil && *r.IDBackKey != ""
}

func (r SellerReg) CanApprove() bool {
	if r.Status != RegPending {
		return false
	}
	if !r.HasIDFront() {
		return false
	}
	if r.IDDocType.NeedsBack() && !r.HasIDBack() {
		return false
	}
	return true
}
