package usecase

import (
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type TOTPSetup struct {
	Secret     string
	OTPAuthURL string
}

func NewTOTPSetup(issuer, account string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, err
	}
	return &TOTPSetup{
		Secret:     key.Secret(),
		OTPAuthURL: key.URL(),
	}, nil
}

func VerifyTOTP(secret, code string) bool {
	return totp.Validate(NormalizeTOTPCode(code), secret)
}

func NormalizeTOTPCode(code string) string {
	var b strings.Builder
	for _, r := range code {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
