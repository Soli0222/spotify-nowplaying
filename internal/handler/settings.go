package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/labstack/echo/v4"
)

// SettingsHandler handles user settings
type SettingsHandler struct {
	store     *store.Store
	jwtConfig auth.JWTConfig
}

// NewSettingsHandler creates a new SettingsHandler
func NewSettingsHandler(s *store.Store, jwtConfig auth.JWTConfig) *SettingsHandler {
	return &SettingsHandler{
		store:     s,
		jwtConfig: jwtConfig,
	}
}

// UserInfoResponse represents the user info response
type UserInfoResponse struct {
	ID            string `json:"id"`
	SpotifyUserID string `json:"spotify_user_id"`

	SpotifyDisplayName string `json:"spotify_display_name,omitempty"`
	SpotifyImageURL    string `json:"spotify_image_url,omitempty"`

	MisskeyConnected   bool   `json:"misskey_connected"`
	MisskeyInstanceURL string `json:"misskey_instance_url,omitempty"`
	MisskeyUserID      string `json:"misskey_user_id,omitempty"`
	MisskeyUsername    string `json:"misskey_username,omitempty"`
	MisskeyAvatarURL   string `json:"misskey_avatar_url,omitempty"`
	MisskeyHost        string `json:"misskey_host,omitempty"`

	TwitterConnected bool   `json:"twitter_connected"`
	TwitterUserID    string `json:"twitter_user_id,omitempty"`
	TwitterUsername  string `json:"twitter_username,omitempty"`
	TwitterAvatarURL string `json:"twitter_avatar_url,omitempty"`

	APIURLToken           string `json:"api_url_token"`
	APIHeaderTokenEnabled bool   `json:"api_header_token_enabled"`
}

// GetUserInfo returns the current user's information
// GET /api/me
func (h *SettingsHandler) GetUserInfo(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()
	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get user"})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	resp := UserInfoResponse{
		ID:                    user.ID.String(),
		SpotifyUserID:         user.SpotifyUserID,
		MisskeyConnected:      user.MisskeyAccessToken.Valid && user.MisskeyAccessToken.String != "",
		TwitterConnected:      user.TwitterAccessToken.Valid && user.TwitterAccessToken.String != "",
		APIURLToken:           user.APIURLToken.String(),
		APIHeaderTokenEnabled: user.APIHeaderTokenEnabled,
	}

	if user.MisskeyInstanceURL.Valid {
		resp.MisskeyInstanceURL = user.MisskeyInstanceURL.String
	}
	if user.MisskeyUserID.Valid {
		resp.MisskeyUserID = user.MisskeyUserID.String
	}
	if user.MisskeyUsername.Valid {
		resp.MisskeyUsername = user.MisskeyUsername.String
	}
	if user.MisskeyAvatarURL.Valid {
		resp.MisskeyAvatarURL = user.MisskeyAvatarURL.String
	}
	if user.MisskeyHost.Valid {
		resp.MisskeyHost = user.MisskeyHost.String
	}

	if user.TwitterUserID.Valid {
		resp.TwitterUserID = user.TwitterUserID.String
	}
	if user.TwitterUsername.Valid {
		resp.TwitterUsername = user.TwitterUsername.String
	}
	if user.TwitterAvatarURL.Valid {
		resp.TwitterAvatarURL = user.TwitterAvatarURL.String
	}

	// Fetch Spotify user profile if access token is available
	if user.SpotifyAccessToken.Valid && user.SpotifyAccessToken.String != "" {
		profile, err := h.getSpotifyUserProfile(user.SpotifyAccessToken.String)
		if err == nil {
			resp.SpotifyDisplayName = profile.DisplayName
			// Use the first (largest) image if available
			if len(profile.Images) > 0 {
				resp.SpotifyImageURL = profile.Images[0].URL
			}
		}
		// Ignore error - just don't show profile info if API call fails
	}

	return c.JSON(http.StatusOK, resp)
}

// getSpotifyUserProfile fetches the user profile from Spotify API
func (h *SettingsHandler) getSpotifyUserProfile(accessToken string) (*SpotifyUserProfile, error) {
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
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

	var profile SpotifyUserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// SpotifyUserProfile represents Spotify user profile
type SpotifyUserProfile struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"display_name"`
	Images      []SpotifyImage `json:"images"`
}

// SpotifyImage represents a Spotify image
type SpotifyImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

// HeaderTokenResponse represents the response after generating a header token
type HeaderTokenResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

// GenerateHeaderToken generates a new header token
// POST /api/settings/header-token
func (h *SettingsHandler) GenerateHeaderToken(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// Generate a new random token
	token, err := auth.GenerateRandomToken(32) // 64 hex characters
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	// Hash the token for storage
	tokenHash := auth.HashToken(token)

	ctx := c.Request().Context()
	if err := h.store.SetAPIHeaderToken(ctx, userID, tokenHash); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save token"})
	}

	// Return the plain token (only shown once)
	return c.JSON(http.StatusOK, HeaderTokenResponse{
		Token:   token,
		Message: "Token generated successfully. Save this token - it will not be shown again.",
	})
}

// DisableHeaderToken disables the header token
// DELETE /api/settings/header-token
func (h *SettingsHandler) DisableHeaderToken(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()
	if err := h.store.DisableAPIHeaderToken(ctx, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to disable token"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "header token disabled"})
}

// RegenerateAPIURLResponse represents the response after regenerating API URL token
type RegenerateAPIURLResponse struct {
	APIURLToken string `json:"api_url_token"`
}

// RegenerateAPIURLToken regenerates the API URL token
// POST /api/settings/api-url-token/regenerate
func (h *SettingsHandler) RegenerateAPIURLToken(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()
	newToken, err := h.store.RegenerateAPIURLToken(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to regenerate token"})
	}

	return c.JSON(http.StatusOK, RegenerateAPIURLResponse{APIURLToken: newToken.String()})
}

// Logout logs out the current user
// POST /api/logout
func (h *SettingsHandler) Logout(c echo.Context) error {
	auth.ClearSessionCookie(c, h.jwtConfig)
	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

// AppConfigResponse represents the app configuration for the frontend
type AppConfigResponse struct {
	TwitterAvailable   bool               `json:"twitter_available"`
	TwitterEligibility TwitterEligibility `json:"twitter_eligibility"`
}

// GetAppConfig returns the app configuration including Twitter eligibility
// GET /api/config
func (h *SettingsHandler) GetAppConfig(c echo.Context) error {
	twitterConfig := LoadTwitterConfig()

	// Check if user is authenticated to determine eligibility
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		// Not authenticated - return basic config
		return c.JSON(http.StatusOK, AppConfigResponse{
			TwitterAvailable:   twitterConfig.IsAvailable(),
			TwitterEligibility: TwitterEligibility{Eligible: false, Reason: "Not authenticated"},
		})
	}

	// Get user info to check Misskey connection
	ctx := c.Request().Context()
	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return c.JSON(http.StatusOK, AppConfigResponse{
			TwitterAvailable:   twitterConfig.IsAvailable(),
			TwitterEligibility: TwitterEligibility{Eligible: false, Reason: "User not found"},
		})
	}

	misskeyConnected := user.MisskeyAccessToken.Valid && user.MisskeyAccessToken.String != ""
	misskeyHost := ""
	if user.MisskeyInstanceURL.Valid {
		misskeyHost = user.MisskeyInstanceURL.String
	}

	eligibility := twitterConfig.CheckEligibility(misskeyConnected, misskeyHost)

	return c.JSON(http.StatusOK, AppConfigResponse{
		TwitterAvailable:   twitterConfig.IsAvailable(),
		TwitterEligibility: eligibility,
	})
}
