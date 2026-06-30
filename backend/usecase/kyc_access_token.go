package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const KYCAccessTokenTTL = 5 * time.Minute

var ErrInvalidKYCToken = errors.New("invalid kyc access token")

func SignKYCAccess(secret []byte, storageKey string, now time.Time) string {
	exp := now.Add(KYCAccessTokenTTL).Unix()
	payload := storageKey + "|" + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
}

func VerifyKYCAccess(secret []byte, token string, now time.Time) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", ErrInvalidKYCToken
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return "", ErrInvalidKYCToken
	}
	key, expStr, sig := parts[0], parts[1], parts[2]
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() > exp {
		return "", ErrInvalidKYCToken
	}
	payload := key + "|" + expStr
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", ErrInvalidKYCToken
	}
	if strings.Contains(key, "..") {
		return "", ErrInvalidKYCToken
	}
	return key, nil
}

func KYCAccessURL(apiBase, token string) string {
	base := strings.TrimRight(apiBase, "/")
	return fmt.Sprintf("%s/admin/kyc-documents?token=%s", base, url.QueryEscape(token))
}
