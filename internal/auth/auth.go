package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
	ErrMissingToken = errors.New("missing token")
)

// Claims represents the JWT claims
type Claims struct {
	UserID        uuid.UUID `json:"user_id"`
	SpotifyUserID string    `json:"spotify_user_id"`
	jwt.RegisteredClaims
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
	CookieName    string
}

// DefaultJWTConfig returns the default JWT configuration
func DefaultJWTConfig() JWTConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-in-production"
	}
	return JWTConfig{
		SecretKey:     secret,
		TokenDuration: 24 * time.Hour * 7, // 7 days
		CookieName:    "session_token",
	}
}

// GenerateToken generates a new JWT token
func GenerateToken(config JWTConfig, userID uuid.UUID, spotifyUserID string) (string, error) {
	claims := &Claims{
		UserID:        userID,
		SpotifyUserID: spotifyUserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.SecretKey))
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(config JWTConfig, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// SetSessionCookie sets the session cookie
func SetSessionCookie(c echo.Context, config JWTConfig, token string) {
	cookie := &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(config.TokenDuration.Seconds()),
	}
	c.SetCookie(cookie)
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(c echo.Context, config JWTConfig) {
	cookie := &http.Cookie{
		Name:     config.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	c.SetCookie(cookie)
}

// GetSessionCookie gets the session token from cookie
func GetSessionCookie(c echo.Context, config JWTConfig) (string, error) {
	cookie, err := c.Cookie(config.CookieName)
	if err != nil {
		return "", ErrMissingToken
	}
	return cookie.Value, nil
}

// JWTMiddleware returns a middleware that validates JWT tokens
func JWTMiddleware(config JWTConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenString, err := GetSessionCookie(c, config)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			claims, err := ValidateToken(config, tokenString)
			if err != nil {
				if errors.Is(err, ErrExpiredToken) {
					ClearSessionCookie(c, config)
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "session expired"})
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid session"})
			}

			// Store claims in context
			c.Set("user_id", claims.UserID)
			c.Set("spotify_user_id", claims.SpotifyUserID)

			return next(c)
		}
	}
}

// GetUserIDFromContext gets the user ID from context
func GetUserIDFromContext(c echo.Context) (uuid.UUID, error) {
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("user_id not found in context")
	}
	return userID, nil
}

// GenerateRandomToken generates a cryptographically secure random token
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken creates a SHA-256 hash of a token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GeneratePKCEVerifier generates a PKCE code verifier
func GeneratePKCEVerifier() (string, error) {
	return GenerateRandomToken(32) // 64 hex characters
}

// GeneratePKCEChallenge generates a PKCE code challenge from verifier (S256)
func GeneratePKCEChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
