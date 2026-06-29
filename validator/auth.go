package validator

import (
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"crowdfund-app/model"
)

func ValidateEmail(email string) (string, error) {
	return model.ValidateEmail(email)
}

func ValidatePassword(password string) error {
	return model.ValidatePassword(password)
}

func ValidateBuyerReg(email, password string) (string, error) {
	return model.ValidateBuyerReg(email, password)
}

func ValidateSellerKYC(fullName string, birthDate time.Time, address string, photoPresent bool) error {
	return model.ValidateSellerKYC(fullName, birthDate, address, photoPresent)
}

const (
	MaxUserNameLen  = 100
	MaxAvatarURLLen = 2048
)

type UpdateProfileInput struct {
	Name      *string
	AvatarURL *string
}

type UpdateProfileNormalized struct {
	Name         *string
	NameSet      bool
	AvatarURL    *string
	AvatarURLSet bool
}

func ValidateUpdateProfile(in UpdateProfileInput) (UpdateProfileNormalized, error) {
	var (
		out UpdateProfileNormalized
		err error
	)
	if in.Name == nil && in.AvatarURL == nil {
		return UpdateProfileNormalized{}, model.ErrBadProfile
	}
	if in.Name != nil {
		out.Name, err = normalizeProfileName(*in.Name)
		if err != nil {
			return UpdateProfileNormalized{}, err
		}
		out.NameSet = true
	}
	if in.AvatarURL != nil {
		out.AvatarURL, err = normalizeAvatarURL(*in.AvatarURL)
		if err != nil {
			return UpdateProfileNormalized{}, err
		}
		out.AvatarURLSet = true
	}
	return out, nil
}

func normalizeProfileName(raw string) (*string, error) {
	var (
		trimmed string
		n       int
	)
	trimmed = strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	n = utf8.RuneCountInString(trimmed)
	if n < 1 || n > MaxUserNameLen {
		return nil, model.ErrBadProfile
	}
	return &trimmed, nil
}

func normalizeAvatarURL(raw string) (*string, error) {
	var (
		trimmed string
		parsed  *url.URL
		err     error
	)
	trimmed = strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > MaxAvatarURLLen {
		return nil, model.ErrBadProfile
	}
	parsed, err = url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, model.ErrBadProfile
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return nil, model.ErrBadProfile
	}
	return &trimmed, nil
}
