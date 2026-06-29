package model

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"column:token_hash;not null;uniqueIndex"` // 平文は DB に保存しない（SHA-256 ハッシュ）
	ExpiresAt time.Time  `gorm:"not null;index"`
	CreatedAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time `gorm:"column:revoked_at"` // NULL = 有効, 値あり = 失効済み
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (t RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t RefreshToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}

type ResetToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"column:token_hash;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"not null"`
}

func (ResetToken) TableName() string { return "password_reset_tokens" }

func (t ResetToken) IsUsed() bool {
	return t.UsedAt != nil
}

func (t ResetToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}

type WSToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"column:token_hash;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"not null"`
}

func (WSToken) TableName() string { return "ws_tokens" }

func (t WSToken) IsUsed() bool {
	return t.UsedAt != nil
}

func (t WSToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}

type TOTPBackupCode struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	CodeHash  string     `gorm:"column:code_hash;not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"not null"`
}

func (TOTPBackupCode) TableName() string { return "totp_backup_codes" }
