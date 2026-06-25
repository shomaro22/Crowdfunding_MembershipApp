package repository

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const kycTokenTTL = 5 * time.Minute

func SignKYCAccess(secret []byte, key string, now time.Time) string {
	var (
		exp     int64
		payload string
		mac     hash.Hash
		sig     string
	)
	exp = now.Add(kycTokenTTL).Unix()
	payload = key + "|" + strconv.FormatInt(exp, 10)
	mac = hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	sig = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + sig))
}

func VerifyKYCAccess(secret []byte, token string, now time.Time) (string, error) {
	var (
		raw      []byte
		parts    []string
		key      string
		expStr   string
		sig      string
		exp      int64
		payload  string
		mac      hash.Hash
		expected string
		err      error
	)
	raw, err = base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", ErrBadToken
	}
	parts = strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return "", ErrBadToken
	}
	key, expStr, sig = parts[0], parts[1], parts[2]
	exp, err = strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() > exp {
		return "", ErrBadToken
	}
	payload = key + "|" + expStr
	mac = hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	expected = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", ErrBadToken
	}
	if strings.Contains(key, "..") {
		return "", ErrBadToken
	}
	return key, nil
}

func KYCAccessURL(apiBase, token string) string {
	var base string
	base = strings.TrimRight(apiBase, "/")
	return fmt.Sprintf("%s/admin/kyc-documents?token=%s", base, url.QueryEscape(token))
}
