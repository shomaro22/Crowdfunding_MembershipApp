package entity

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
	MinProjectTitle   = 1
	MaxProjectTitle   = 200
	MinPlanPriceJPY   = 1
	MaxPlanPriceJPY   = 100_000
	MaxProjectImages  = 10
)

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func ValidateEmail(email string) (string, error) {
	normalized := NormalizeEmail(email)
	if normalized == "" {
		return "", ErrBadEmail
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", ErrBadEmail
	}
	return normalized, nil
}

func ValidatePassword(password string) error {
	if runeLen(password) < MinPasswordLength {
		return ErrWeakPassword
	}
	if runeLen(password) > MaxPasswordLength {
		return ErrWeakPassword
	}
	return nil
}

func ValidateBuyerReg(email, password string) (string, error) {
	normalized, err := ValidateEmail(email)
	if err != nil {
		return "", err
	}
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	return normalized, nil
}

func ValidateTitle(title string) error {
	n := runeLen(strings.TrimSpace(title))
	if n < MinProjectTitle || n > MaxProjectTitle {
		return ErrBadTitle
	}
	return nil
}

func ValidatePrice(price int) error {
	if price < MinPlanPriceJPY || price > MaxPlanPriceJPY {
		return ErrBadPrice
	}
	return nil
}

func ValidateMessage(content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ErrBadMessage
	}
	if runeLen(trimmed) > MaxMessageLength {
		return ErrBadMessage
	}
	return nil
}

// ValidateSellerKYC は免許証照合用メタデータを検証する。
func ValidateSellerKYC(fullName string, birthDate time.Time, address string, photoPresent bool) error {
	if strings.TrimSpace(fullName) == "" {
		return ErrBadKYC
	}
	if birthDate.IsZero() {
		return ErrBadKYC
	}
	if strings.TrimSpace(address) == "" {
		return ErrBadKYC
	}
	if !photoPresent {
		return ErrBadKYC
	}
	return nil
}

func ValidateSellerIDDoc(docType IDDocType) error {
	if !docType.Valid() {
		return ErrBadIDDoc
	}
	return nil
}

func ValidateDuration(duration DurationType, start time.Time, end *time.Time) error {
	if !duration.Valid() {
		return ErrBadDuration
	}
	if duration == DurationLimited {
		if end == nil || !end.After(start) {
			return ErrBadDuration
		}
	}
	if duration == DurationUnlimited && end != nil {
		return ErrBadDuration
	}
	return nil
}
