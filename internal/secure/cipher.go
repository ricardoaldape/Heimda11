package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

type Cipher struct {
	aead cipher.AEAD
}

func NewCipher(hexKey string) (*Cipher, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, errors.New("HEIMDA11_MASTER_KEY must be exactly 64 hexadecimal characters")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (c *Cipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(raw) < c.aead.NonceSize() {
		return "", errors.New("ciphertext is too short")
	}
	nonce := raw[:c.aead.NonceSize()]
	ciphertext := raw[c.aead.NonceSize():]
	plain, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}