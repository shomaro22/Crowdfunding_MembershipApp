package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

const backupCodeCount = 10

func (d *authDeps) verifyTOTPOrBackup(ctx context.Context, u model.User, code string) bool {
	var (
		secret string
		err    error
		norm   string
		rows   []model.TOTPBackupCode
		hash   string
		now    time.Time
		i      int
		row    model.TOTPBackupCode
	)

	secret, err = d.decryptTOTP(u)
	if err == nil && VerifyTOTP(secret, code) {
		return true
	}
	if d.TOTPBackup == nil {
		return false
	}
	norm = strings.ToUpper(strings.TrimSpace(code))
	if norm == "" {
		return false
	}
	rows, err = d.TOTPBackup.ListUnused(ctx, u.ID)
	if err != nil {
		return false
	}
	hash = OpaqueHash(norm)
	now = time.Now().UTC()
	for i = range rows {
		row = rows[i]
		if row.CodeHash == hash {
			_ = d.TOTPBackup.MarkUsed(ctx, row.ID, now)
			return true
		}
	}
	return false
}

func (d *authDeps) decryptTOTP(u model.User) (string, error) {
	var (
		plain string
		err   error
	)

	if u.TOTPSecret == nil || strings.TrimSpace(*u.TOTPSecret) == "" {
		return "", Err2FANotConfigured
	}
	plain, err = DecryptSecret(d.TOTPKey, *u.TOTPSecret)
	if err == nil {
		return plain, nil
	}
	if len(d.legacyTOTPKey) >= 32 && !bytes.Equal(d.legacyTOTPKey, d.TOTPKey) {
		return DecryptSecret(d.legacyTOTPKey, *u.TOTPSecret)
	}
	return "", err
}

func (d *authDeps) genBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var (
		now   time.Time
		plain []string
		rows  []model.TOTPBackupCode
		i     int
		code  string
		err   error
	)

	if d.TOTPBackup == nil {
		return nil, nil
	}
	_ = d.TOTPBackup.DeleteUser(ctx, userID)
	now = time.Now().UTC()
	plain = make([]string, 0, backupCodeCount)
	rows = make([]model.TOTPBackupCode, 0, backupCodeCount)
	for i = 0; i < backupCodeCount; i++ {
		code, err = randBackupCode()
		if err != nil {
			return nil, err
		}
		plain = append(plain, code)
		rows = append(rows, model.TOTPBackupCode{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHash:  OpaqueHash(code),
			CreatedAt: now,
		})
	}
	if err = d.TOTPBackup.CreateBatch(ctx, rows); err != nil {
		return nil, err
	}
	return plain, nil
}

func randBackupCode() (string, error) {
	var (
		b   []byte
		err error
	)

	b = make([]byte, 4)
	if _, err = rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}
