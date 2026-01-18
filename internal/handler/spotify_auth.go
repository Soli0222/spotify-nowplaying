package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/spotify"
	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/labstack/echo/v4"
)

// SpotifyAuthHandler handles Spotify OAuth authentication for the frontend
type SpotifyAuthHandler struct {
	store         *store.Store
	spotifyClient spotify.Client
	jwtConfig     auth.JWTConfig
}

// NewSpotifyAuthHandler creates a new SpotifyAuthHandler
func NewSpotifyAuthHandler(s *store.Store, client spotify.Client, jwtConfig auth.JWTConfig) *SpotifyAuthHandler {
	return &SpotifyAuthHandler{
		store:         s,
		spotifyClient: client,
		jwtConfig:     jwtConfig,
	}
}

// SpotifyUserResponse represents the Spotify user profile response
type SpotifyUserResponse struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"display_name"`
	Email       string         `json:"email"`
	Images      []SpotifyImage `json:"images"`
}

// LoginSpotify redirects to Spotify OAuth
// GET /api/auth/spotify
func (h *SpotifyAuthHandler) LoginSpotify(c echo.Context) error {
	authURL := "https://accounts.spotify.com/authorize"
	scope := "user-read-currently-playing user-read-playback-state"
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	redirectURI := os.Getenv("BASE_URL") + "/api/auth/spotify/callback"

	redirectURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		authURL, clientID, redirectURI, scope)

	return c.Redirect(http.StatusFound, redirectURL)
}

// CallbackSpotify handles the Spotify OAuth callback
// GET /api/auth/spotify/callback
func (h *SpotifyAuthHandler) CallbackSpotify(c echo.Context) error {
	code := c.QueryParam("code")
	errorParam := c.QueryParam("error")

	if errorParam != "" {
		return c.Redirect(http.StatusFound, "/login?error=spotify_auth_denied")
	}

	if code == "" {
		return c.Redirect(http.StatusFound, "/login?error=missing_code")
	}

	redirectURI := os.Getenv("BASE_URL") + "/api/auth/spotify/callback"

	// Exchange code for tokens
	tokens, err := h.spotifyClient.ExchangeToken(code, redirectURI)
	if err != nil {
		return c.Redirect(http.StatusFound, "/login?error=token_exchange_failed")
	}

	// Get user profile from Spotify
	userProfile, err := h.getSpotifyUserProfile(tokens.AccessToken)
	if err != nil {
		return c.Redirect(http.StatusFound, "/login?error=profile_fetch_failed")
	}

	ctx := c.Request().Context()

	// Calculate token expiration
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Create or update user in database
	user, err := h.store.CreateUser(ctx, userProfile.ID, tokens.AccessToken, tokens.RefreshToken, expiresAt)
	if err != nil {
		return c.Redirect(http.StatusFound, "/login?error=user_creation_failed")
	}

	// Generate JWT token
	jwtToken, err := auth.GenerateToken(h.jwtConfig, user.ID, user.SpotifyUserID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/login?error=jwt_generation_failed")
	}

	// Set session cookie
	auth.SetSessionCookie(c, h.jwtConfig, jwtToken)

	return c.Redirect(http.StatusFound, "/dashboard")
}

// getSpotifyUserProfile fetches the user profile from Spotify
func (h *SpotifyAuthHandler) getSpotifyUserProfile(accessToken string) (*SpotifyUserResponse, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify api error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userProfile SpotifyUserResponse
	if err := json.Unmarshal(body, &userProfile); err != nil {
		return nil, err
	}

	return &userProfile, nil
}

// CheckAuth checks if the user is authenticated
// GET /api/auth/check
func (h *SpotifyAuthHandler) CheckAuth(c echo.Context) error {
	tokenString, err := auth.GetSessionCookie(c, h.jwtConfig)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
	}

	claims, err := auth.ValidateToken(h.jwtConfig, tokenString)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"authenticated":   true,
		"user_id":         claims.UserID.String(),
		"spotify_user_id": claims.SpotifyUserID,
	})
}
