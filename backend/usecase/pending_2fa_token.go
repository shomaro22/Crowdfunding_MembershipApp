package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const pending2FATTL = 5 * time.Minute

func SignPending2FA(secret []byte, userID uuid.UUID, now time.Time) string {
	exp := now.Add(pending2FATTL).Unix()
	payload := userID.String() + "|" + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
}

func VerifyPending2FA(secret []byte, token string, now time.Time) (uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return uuid.Nil, ErrInvalidToken
	}
	uidStr, expStr, sig := parts[0], parts[1], parts[2]
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() > exp {
		return uuid.Nil, ErrInvalidToken
	}
	payload := uidStr + "|" + expStr
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return uuid.Nil, ErrInvalidToken
	}
	return uuid.Parse(uidStr)
}
