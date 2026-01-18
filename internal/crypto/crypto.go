package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("encryption key must be 32 bytes (256 bits)")
	// ErrInvalidCiphertext is returned when the ciphertext is invalid
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrEncryptionNotConfigured is returned when encryption is not configured
	ErrEncryptionNotConfigured = errors.New("encryption not configured: TOKEN_ENCRYPTION_KEY not set")
)

// TokenEncryptor handles encryption and decryption of tokens
type TokenEncryptor struct {
	gcm cipher.AEAD
}

var (
	defaultEncryptor *TokenEncryptor
	encryptorOnce    sync.Once
	encryptorErr     error
)

// GetDefaultEncryptor returns the default token encryptor
// It initializes the encryptor from TOKEN_ENCRYPTION_KEY environment variable
func GetDefaultEncryptor() (*TokenEncryptor, error) {
	encryptorOnce.Do(func() {
		key := os.Getenv("TOKEN_ENCRYPTION_KEY")
		if key == "" {
			encryptorErr = ErrEncryptionNotConfigured
			return
		}

		defaultEncryptor, encryptorErr = NewTokenEncryptor([]byte(key))
	})

	return defaultEncryptor, encryptorErr
}

// NewTokenEncryptor creates a new TokenEncryptor with the given key
// The key must be exactly 32 bytes for AES-256
func NewTokenEncryptor(key []byte) (*TokenEncryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &TokenEncryptor{gcm: gcm}, nil
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext
func (e *TokenEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptToken encrypts a token using the default encryptor
// Returns the original token if encryption is not configured
func EncryptToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}

	encryptor, err := GetDefaultEncryptor()
	if err != nil {
		if errors.Is(err, ErrEncryptionNotConfigured) {
			// Return original token if encryption is not configured
			return token, nil
		}
		return "", err
	}

	return encryptor.Encrypt(token)
}

// DecryptToken decrypts a token using the default encryptor
// Attempts to decrypt, but returns the original value if decryption fails
// (for backward compatibility with unencrypted tokens)
func DecryptToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}

	encryptor, err := GetDefaultEncryptor()
	if err != nil {
		if errors.Is(err, ErrEncryptionNotConfigured) {
			// Return original token if encryption is not configured
			return token, nil
		}
		return "", err
	}

	decrypted, err := encryptor.Decrypt(token)
	if err != nil {
		// If decryption fails, assume it's an unencrypted token (backward compatibility)
		return token, nil
	}

	return decrypted, nil
}

// GenerateKey generates a random 32-byte encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 generates a random 32-byte encryption key and returns it as base64
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
