package trading212

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// ErrNoEncryptionKey is returned when CREDENTIALS_ENCRYPTION_KEY is unset,
// so credentials can be neither stored nor read.
var ErrNoEncryptionKey = errors.New("CREDENTIALS_ENCRYPTION_KEY is not configured")

// Cipher encrypts API credentials at rest with AES-256-GCM. A nil *Cipher
// (no key configured) refuses every operation with ErrNoEncryptionKey.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher builds a Cipher from a base64-encoded 32-byte key. An empty key
// returns (nil, nil): the feature is disabled rather than the app failing.
func NewCipher(base64Key string) (*Cipher, error) {
	base64Key = strings.TrimSpace(base64Key)
	if base64Key == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("decoding CREDENTIALS_ENCRYPTION_KEY: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("CREDENTIALS_ENCRYPTION_KEY must decode to 32 bytes, got %d", len(key))
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

// Encrypt seals plaintext, binding it to associatedData (the account id) so
// a ciphertext copied onto another account's row fails to decrypt.
func (c *Cipher) Encrypt(plaintext, associatedData []byte) ([]byte, error) {
	if c == nil {
		return nil, ErrNoEncryptionKey
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, plaintext, associatedData), nil
}

func (c *Cipher) Decrypt(ciphertext, associatedData []byte) ([]byte, error) {
	if c == nil {
		return nil, ErrNoEncryptionKey
	}
	n := c.aead.NonceSize()
	if len(ciphertext) < n {
		return nil, errors.New("ciphertext too short")
	}
	plaintext, err := c.aead.Open(nil, ciphertext[:n], ciphertext[n:], associatedData)
	if err != nil {
		return nil, errors.New("decrypting credentials failed (wrong CREDENTIALS_ENCRYPTION_KEY?)")
	}
	return plaintext, nil
}
