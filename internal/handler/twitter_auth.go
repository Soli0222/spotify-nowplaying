package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/labstack/echo/v4"
)

// TwitterAuthHandler handles Twitter OAuth 2.0 PKCE authentication
type TwitterAuthHandler struct {
	store     *store.Store
	jwtConfig auth.JWTConfig
}

// NewTwitterAuthHandler creates a new TwitterAuthHandler
func NewTwitterAuthHandler(s *store.Store, jwtConfig auth.JWTConfig) *TwitterAuthHandler {
	return &TwitterAuthHandler{
		store:     s,
		jwtConfig: jwtConfig,
	}
}

// TwitterTokenResponse is the response from Twitter token endpoint
type TwitterTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// StartTwitterAuth starts the Twitter OAuth 2.0 PKCE flow
// GET /api/twitter/start
func (h *TwitterAuthHandler) StartTwitterAuth(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// Check Twitter eligibility
	twitterConfig := LoadTwitterConfig()
	if !twitterConfig.IsAvailable() {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Twitter integration is not available"})
	}

	// Get user to check Misskey connection for eligibility
	ctx := c.Request().Context()
	user, err := h.store.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get user"})
	}

	misskeyConnected := user.MisskeyAccessToken.Valid && user.MisskeyAccessToken.String != ""
	misskeyHost := ""
	if user.MisskeyInstanceURL.Valid {
		misskeyHost = user.MisskeyInstanceURL.String
	}

	eligibility := twitterConfig.CheckEligibility(misskeyConnected, misskeyHost)
	if !eligibility.Eligible {
		return c.JSON(http.StatusForbidden, map[string]string{"error": eligibility.Reason})
	}

	// Generate PKCE verifier and challenge
	verifier, err := auth.GeneratePKCEVerifier()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate verifier"})
	}
	challenge := auth.GeneratePKCEChallenge(verifier)

	// Generate state
	state, err := auth.GenerateRandomToken(16)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate state"})
	}

	// Store the session
	if err := h.store.CreateTwitterPKCESession(ctx, userID, state, verifier); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	// Build Twitter OAuth URL
	clientID := os.Getenv("TWITTER_CLIENT_ID")
	redirectURI := os.Getenv("BASE_URL") + "/api/twitter/callback"
	scope := "tweet.read tweet.write users.read offline.access"

	authURL := fmt.Sprintf(
		"https://x.com/i/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scope),
		url.QueryEscape(state),
		url.QueryEscape(challenge),
	)

	return c.Redirect(http.StatusFound, authURL)
}

// CallbackTwitterAuth handles the Twitter OAuth callback
// GET /api/twitter/callback
func (h *TwitterAuthHandler) CallbackTwitterAuth(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")
	errorParam := c.QueryParam("error")

	if errorParam != "" {
		return c.Redirect(http.StatusFound, "/dashboard?error=twitter_auth_denied")
	}

	if code == "" || state == "" {
		return c.Redirect(http.StatusFound, "/dashboard?error=missing_params")
	}

	ctx := c.Request().Context()

	// Get the session
	session, err := h.store.GetTwitterPKCESession(ctx, state)
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=session_error")
	}
	if session == nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=session_not_found")
	}

	// Exchange code for tokens
	clientID := os.Getenv("TWITTER_CLIENT_ID")
	clientSecret := os.Getenv("TWITTER_CLIENT_SECRET")
	redirectURI := os.Getenv("BASE_URL") + "/api/twitter/callback"

	data := url.Values{}
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", session.CodeVerifier)

	req, err := http.NewRequest("POST", "https://api.twitter.com/2/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=request_failed")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=exchange_failed")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=read_failed")
	}

	if resp.StatusCode != http.StatusOK {
		return c.Redirect(http.StatusFound, "/dashboard?error=token_failed")
	}

	var tokenResp TwitterTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=parse_failed")
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Fetch Twitter user info
	twitterUser, err := h.getTwitterUserInfo(tokenResp.AccessToken)
	if err != nil {
		// If we can't get user info, still save the token but with empty user info
		twitterUser = &TwitterUserInfo{}
	}

	// Save the token and user info to user
	if err := h.store.UpdateTwitterToken(ctx, session.UserID, tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt, twitterUser.ID, twitterUser.Username, twitterUser.ProfileImageURL); err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=save_failed")
	}

	// Delete the session (ignore error - session cleanup is best effort)
	_ = h.store.DeleteTwitterPKCESession(ctx, state)

	return c.Redirect(http.StatusFound, "/dashboard?success=twitter_connected")
}

// RefreshTwitterToken refreshes the Twitter access token
func (h *TwitterAuthHandler) RefreshTwitterToken(ctx echo.Context, userID string, refreshToken string) (*TwitterTokenResponse, error) {
	clientID := os.Getenv("TWITTER_CLIENT_ID")
	clientSecret := os.Getenv("TWITTER_CLIENT_SECRET")

	data := url.Values{}
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", "https://api.twitter.com/2/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed: %s", string(body))
	}

	var tokenResp TwitterTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// DisconnectTwitter disconnects Twitter from the user account
// DELETE /api/twitter
func (h *TwitterAuthHandler) DisconnectTwitter(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()
	if err := h.store.DisconnectTwitter(ctx, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to disconnect"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "twitter disconnected"})
}

// TwitterUserInfo represents Twitter user information
type TwitterUserInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

// TwitterUserResponse represents the response from Twitter users/me endpoint
type TwitterUserResponse struct {
	Data TwitterUserInfo `json:"data"`
}

// getTwitterUserInfo fetches the Twitter user information
func (h *TwitterAuthHandler) getTwitterUserInfo(accessToken string) (*TwitterUserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.twitter.com/2/users/me?user.fields=profile_image_url", nil)
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
		return nil, fmt.Errorf("twitter api error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userResp TwitterUserResponse
	if err := json.Unmarshal(body, &userResp); err != nil {
		return nil, err
	}

	return &userResp.Data, nil
}
