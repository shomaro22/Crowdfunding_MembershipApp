package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func OpaqueHex() (string, error) {
	var (
		b   []byte
		err error
	)
	b = make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func OpaqueHash(plain string) string {
	var sum [32]byte
	sum = sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
