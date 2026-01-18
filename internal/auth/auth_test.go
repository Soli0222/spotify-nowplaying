package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()

	assert.NotEmpty(t, config.SecretKey)
	assert.Equal(t, 24*time.Hour*7, config.TokenDuration)
	assert.Equal(t, "session_token", config.CookieName)
}

func TestDefaultJWTConfig_WithEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-test-secret")

	config := DefaultJWTConfig()

	assert.Equal(t, "my-test-secret", config.SecretKey)
}

func TestGenerateToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}
	userID := uuid.New()
	spotifyUserID := "spotify123"

	token, err := GenerateToken(config, userID, spotifyUserID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_Success(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}
	userID := uuid.New()
	spotifyUserID := "spotify123"

	token, err := GenerateToken(config, userID, spotifyUserID)
	require.NoError(t, err)

	claims, err := ValidateToken(config, token)

	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, spotifyUserID, claims.SpotifyUserID)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	_, err := ValidateToken(config, "invalid-token")

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	config1 := JWTConfig{
		SecretKey:     "secret1",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}
	config2 := JWTConfig{
		SecretKey:     "secret2",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}
	userID := uuid.New()

	token, err := GenerateToken(config1, userID, "spotify123")
	require.NoError(t, err)

	_, err = ValidateToken(config2, token)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: -time.Hour, // 過去のトークン
		CookieName:    "session_token",
	}
	userID := uuid.New()

	token, err := GenerateToken(config, userID, "spotify123")
	require.NoError(t, err)

	_, err = ValidateToken(config, token)

	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestSetSessionCookie(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	SetSessionCookie(c, config, "test-token")

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "session_token", cookies[0].Name)
	assert.Equal(t, "test-token", cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
}

func TestClearSessionCookie(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ClearSessionCookie(c, config)

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "session_token", cookies[0].Name)
	assert.Equal(t, "", cookies[0].Value)
	assert.Equal(t, -1, cookies[0].MaxAge)
}

func TestGetSessionCookie_Success(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "test-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	token, err := GetSessionCookie(c, config)

	require.NoError(t, err)
	assert.Equal(t, "test-token", token)
}

func TestGetSessionCookie_Missing(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := GetSessionCookie(c, config)

	assert.ErrorIs(t, err, ErrMissingToken)
}

func TestJWTMiddleware_Success(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}
	userID := uuid.New()
	token, err := GenerateToken(config, userID, "spotify123")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	handler := func(c echo.Context) error {
		handlerCalled = true
		contextUserID, err := GetUserIDFromContext(c)
		assert.NoError(t, err)
		assert.Equal(t, userID, contextUserID)
		return c.NoContent(http.StatusOK)
	}

	middleware := JWTMiddleware(config)
	err = middleware(handler)(c)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestJWTMiddleware_NoCookie(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	middleware := JWTMiddleware(config)
	err := middleware(handler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "invalid-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	middleware := JWTMiddleware(config)
	err := middleware(handler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	configForGenerate := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: -time.Hour, // 期限切れトークン生成用
		CookieName:    "session_token",
	}
	configForValidate := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}

	userID := uuid.New()
	token, err := GenerateToken(configForGenerate, userID, "spotify123")
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	middleware := JWTMiddleware(configForValidate)
	err = middleware(handler)(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	// セッション期限切れ時はクッキーがクリアされる
	cookies := rec.Result().Cookies()
	var clearedCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			clearedCookie = cookie
			break
		}
	}
	require.NotNil(t, clearedCookie)
	assert.Equal(t, -1, clearedCookie.MaxAge)
}

func TestGetUserIDFromContext_Success(t *testing.T) {
	userID := uuid.New()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID)

	result, err := GetUserIDFromContext(c)

	require.NoError(t, err)
	assert.Equal(t, userID, result)
}

func TestGetUserIDFromContext_Missing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := GetUserIDFromContext(c)

	assert.Error(t, err)
}

func TestGetUserIDFromContext_WrongType(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "not-a-uuid")

	_, err := GetUserIDFromContext(c)

	assert.Error(t, err)
}

func TestGenerateRandomToken(t *testing.T) {
	token1, err := GenerateRandomToken(32)
	require.NoError(t, err)
	assert.Len(t, token1, 64) // hexエンコードで2倍

	token2, err := GenerateRandomToken(32)
	require.NoError(t, err)

	// 2つのトークンは異なるべき
	assert.NotEqual(t, token1, token2)
}

func TestGenerateRandomToken_DifferentLengths(t *testing.T) {
	testCases := []struct {
		length         int
		expectedLength int
	}{
		{16, 32},
		{32, 64},
		{64, 128},
	}

	for _, tc := range testCases {
		token, err := GenerateRandomToken(tc.length)
		require.NoError(t, err)
		assert.Len(t, token, tc.expectedLength)
	}
}

func TestHashToken(t *testing.T) {
	token := "my-secret-token"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	// 同じ入力は同じハッシュになる
	assert.Equal(t, hash1, hash2)
	// SHA-256は64文字のhex
	assert.Len(t, hash1, 64)

	// 異なる入力は異なるハッシュになる
	hash3 := HashToken("different-token")
	assert.NotEqual(t, hash1, hash3)
}

func TestHashToken_EmptyString(t *testing.T) {
	hash := HashToken("")
	assert.Len(t, hash, 64)
}

func TestGeneratePKCEVerifier(t *testing.T) {
	verifier1, err := GeneratePKCEVerifier()
	require.NoError(t, err)
	assert.Len(t, verifier1, 64)

	verifier2, err := GeneratePKCEVerifier()
	require.NoError(t, err)

	// 2つのverifierは異なるべき
	assert.NotEqual(t, verifier1, verifier2)
}

func TestGeneratePKCEChallenge(t *testing.T) {
	verifier := "test-verifier"

	challenge := GeneratePKCEChallenge(verifier)

	assert.NotEmpty(t, challenge)
	// 同じverifierは同じchallengeになる
	assert.Equal(t, challenge, GeneratePKCEChallenge(verifier))
	// 異なるverifierは異なるchallengeになる
	assert.NotEqual(t, challenge, GeneratePKCEChallenge("other-verifier"))
}

func TestGeneratePKCEChallenge_Base64URLEncoded(t *testing.T) {
	verifier := "test-verifier-12345"

	challenge := GeneratePKCEChallenge(verifier)

	// Base64URLエンコードなので、+, /, = は含まれない
	assert.NotContains(t, challenge, "+")
	assert.NotContains(t, challenge, "/")
	assert.NotContains(t, challenge, "=")
}

func TestClaims_RoundTrip(t *testing.T) {
	config := JWTConfig{
		SecretKey:     "test-secret",
		TokenDuration: time.Hour,
		CookieName:    "session_token",
	}
	originalUserID := uuid.New()
	originalSpotifyID := "spotify-user-123"

	token, err := GenerateToken(config, originalUserID, originalSpotifyID)
	require.NoError(t, err)

	claims, err := ValidateToken(config, token)
	require.NoError(t, err)

	assert.Equal(t, originalUserID, claims.UserID)
	assert.Equal(t, originalSpotifyID, claims.SpotifyUserID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.NotNil(t, claims.NotBefore)
}
