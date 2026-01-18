package crypto

import (
	"encoding/base64"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenEncryptor_ValidKey(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewTokenEncryptor(key)
	require.NoError(t, err)
	assert.NotNil(t, encryptor)
}

func TestNewTokenEncryptor_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name   string
		keyLen int
	}{
		{"too short", 16},
		{"too long", 64},
		{"empty", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewTokenEncryptor(key)
			assert.ErrorIs(t, err, ErrInvalidKey)
		})
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := []byte("12345678901234567890123456789012") // 32 bytes
	encryptor, err := NewTokenEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple token", "access_token_12345"},
		{"long token", strings.Repeat("a", 1000)},
		{"special characters", "token!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "トークン_токен_🔐"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := encryptor.Encrypt(tt.plaintext)
			require.NoError(t, err)

			if tt.plaintext == "" {
				assert.Empty(t, encrypted)
				return
			}

			// Encrypted should be different from plaintext
			assert.NotEqual(t, tt.plaintext, encrypted)

			// Should be base64 encoded
			_, err = base64.StdEncoding.DecodeString(encrypted)
			assert.NoError(t, err)

			// Decrypt should return original
			decrypted, err := encryptor.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	encryptor, err := NewTokenEncryptor(key)
	require.NoError(t, err)

	plaintext := "same_token"
	encrypted1, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	// Same plaintext should produce different ciphertext (due to random nonce)
	assert.NotEqual(t, encrypted1, encrypted2)

	// Both should decrypt to the same plaintext
	decrypted1, err := encryptor.Decrypt(encrypted1)
	require.NoError(t, err)
	decrypted2, err := encryptor.Decrypt(encrypted2)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted1)
	assert.Equal(t, plaintext, decrypted2)
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	encryptor, err := NewTokenEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"not base64", "not-valid-base64!!!"},
		{"too short", base64.StdEncoding.EncodeToString([]byte("short"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.ciphertext)
			assert.Error(t, err)
		})
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := []byte("12345678901234567890123456789012")
	key2 := []byte("abcdefghijklmnopqrstuvwxyz123456")

	encryptor1, err := NewTokenEncryptor(key1)
	require.NoError(t, err)

	encryptor2, err := NewTokenEncryptor(key2)
	require.NoError(t, err)

	encrypted, err := encryptor1.Encrypt("secret_token")
	require.NoError(t, err)

	// Decrypting with wrong key should fail
	_, err = encryptor2.Decrypt(encrypted)
	assert.Error(t, err)
}

func TestEncryptToken_WithEnv(t *testing.T) {
	// Reset the singleton for testing
	encryptorOnce = sync.Once{}
	defaultEncryptor = nil
	encryptorErr = nil

	t.Setenv("TOKEN_ENCRYPTION_KEY", "12345678901234567890123456789012")

	token := "my_secret_token"
	encrypted, err := EncryptToken(token)
	require.NoError(t, err)
	assert.NotEqual(t, token, encrypted)

	decrypted, err := DecryptToken(encrypted)
	require.NoError(t, err)
	assert.Equal(t, token, decrypted)
}

func TestEncryptToken_WithoutEnv(t *testing.T) {
	// Reset the singleton for testing
	encryptorOnce = sync.Once{}
	defaultEncryptor = nil
	encryptorErr = nil

	// Unset the environment variable
	t.Setenv("TOKEN_ENCRYPTION_KEY", "")

	token := "my_secret_token"
	result, err := EncryptToken(token)
	require.NoError(t, err)
	// Should return original token when encryption is not configured
	assert.Equal(t, token, result)
}

func TestDecryptToken_BackwardCompatibility(t *testing.T) {
	// Reset the singleton for testing
	encryptorOnce = sync.Once{}
	defaultEncryptor = nil
	encryptorErr = nil

	t.Setenv("TOKEN_ENCRYPTION_KEY", "12345678901234567890123456789012")

	// Unencrypted token should be returned as-is
	unencryptedToken := "plain_text_token"
	result, err := DecryptToken(unencryptedToken)
	require.NoError(t, err)
	assert.Equal(t, unencryptedToken, result)
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key1, 32)

	key2, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key2, 32)

	// Keys should be different (random)
	assert.NotEqual(t, key1, key2)
}

func TestGenerateKeyBase64(t *testing.T) {
	keyBase64, err := GenerateKeyBase64()
	require.NoError(t, err)

	// Should be valid base64
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestEmptyToken(t *testing.T) {
	// Reset the singleton for testing
	encryptorOnce = sync.Once{}
	defaultEncryptor = nil
	encryptorErr = nil

	t.Setenv("TOKEN_ENCRYPTION_KEY", "12345678901234567890123456789012")

	encrypted, err := EncryptToken("")
	require.NoError(t, err)
	assert.Empty(t, encrypted)

	decrypted, err := DecryptToken("")
	require.NoError(t, err)
	assert.Empty(t, decrypted)
}
