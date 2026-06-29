package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

func EncryptSecret(key []byte, plain string) (string, error) {
	var (
		k     []byte
		block cipher.Block
		gcm   cipher.AEAD
		nonce []byte
		out   []byte
		err   error
	)
	k = deriveAESKey(key)
	block, err = aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce = make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}
	out = gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

func DecryptSecret(key []byte, enc string) (string, error) {
	var (
		raw   []byte
		k     []byte
		block cipher.Block
		gcm   cipher.AEAD
		nonce []byte
		ct    []byte
		plain []byte
		err   error
	)
	raw, err = base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", ErrInvalidToken
	}
	k = deriveAESKey(key)
	block, err = aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	gcm, err = cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrInvalidToken
	}
	nonce, ct = raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err = gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", ErrInvalidToken
	}
	return string(plain), nil
}

func deriveAESKey(key []byte) []byte {
	var (
		out []byte
		i   int
	)
	out = make([]byte, 32)
	copy(out, key)
	if len(key) >= 32 {
		return key[:32]
	}
	for i = 32; i < len(key); i++ {
		out[i%32] ^= key[i]
	}
	return out
}
