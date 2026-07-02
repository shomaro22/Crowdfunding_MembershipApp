package model

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
	var (
		normalized string
		err        error
	)
	normalized = NormalizeEmail(email)
	if normalized == "" {
		return "", ErrBadEmail
	}
	_, err = mail.ParseAddress(normalized)
	if err != nil {
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
	var (
		normalized string
		err        error
	)
	normalized, err = ValidateEmail(email)
	if err != nil {
		return "", err
	}
	err = ValidatePassword(password)
	if err != nil {
		return "", err
	}
	return normalized, nil
}

func ValidateTitle(title string) error {
	var n int
	n = runeLen(strings.TrimSpace(title))
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
	var trimmed string
	trimmed = strings.TrimSpace(content)
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

type PlanInput struct {
	MembershipType MembershipType
	Price          int
	MaxSupporters  *int
	Benefits       string
}

type ImageInput struct {
	ImageURL string
	IsMain   bool
	Order    int
}

func ValidateProjectImages(images []ImageInput) error {
	var (
		mainCount int
		img       ImageInput
	)
	if len(images) > MaxProjectImages {
		return ErrBadImages
	}
	for _, img = range images {
		if strings.TrimSpace(img.ImageURL) == "" {
			return ErrBadImages
		}
		if img.IsMain {
			mainCount++
		}
	}
	if len(images) > 0 && mainCount != 1 {
		return ErrBadImages
	}
	return nil
}

func ValidateProject(title, description string, targetAmount int64, duration DurationType, start time.Time, end *time.Time, plans []PlanInput) error {
	var (
		seen map[MembershipType]bool
		p    PlanInput
		err  error
	)
	err = ValidateTitle(title)
	if err != nil {
		return err
	}
	if strings.TrimSpace(description) == "" {
		return ErrBadProject
	}
	if targetAmount < 0 {
		return ErrBadProject
	}
	err = ValidateDuration(duration, start, end)
	if err != nil {
		return err
	}
	if len(plans) < 1 || len(plans) > 3 {
		return ErrBadProject
	}
	seen = map[MembershipType]bool{}
	for _, p = range plans {
		if !p.MembershipType.Valid() {
			return ErrBadMembership
		}
		if seen[p.MembershipType] {
			return ErrBadProject
		}
		seen[p.MembershipType] = true
		if strings.TrimSpace(p.Benefits) == "" {
			return ErrBadProject
		}
		err = ValidatePrice(p.Price)
		if err != nil {
			return err
		}
	}
	return nil
}
