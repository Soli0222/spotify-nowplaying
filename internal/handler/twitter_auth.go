package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/Soli0222/spotify-nowplaying/internal/twitter"
	"github.com/labstack/echo/v4"
)

// TwitterAuthHandler handles Twitter OAuth 2.0 PKCE authentication
type TwitterAuthHandler struct {
	store         *store.Store
	twitterClient twitter.Client
	jwtConfig     auth.JWTConfig
}

// NewTwitterAuthHandler creates a new TwitterAuthHandler
func NewTwitterAuthHandler(s *store.Store, twitterClient twitter.Client, jwtConfig auth.JWTConfig) *TwitterAuthHandler {
	return &TwitterAuthHandler{
		store:         s,
		twitterClient: twitterClient,
		jwtConfig:     jwtConfig,
	}
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
	redirectURI := os.Getenv("BASE_URL") + "/api/twitter/callback"
	tokens, err := h.twitterClient.ExchangeToken(ctx, code, redirectURI, session.CodeVerifier)
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=token_failed")
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Fetch Twitter user info
	twitterUser, err := h.twitterClient.GetUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		// If we can't get user info, still save the token but with empty user info
		twitterUser = &twitter.UserInfo{}
	}

	// Save the token and user info to user
	if err := h.store.UpdateTwitterToken(ctx, session.UserID, tokens.AccessToken, tokens.RefreshToken, expiresAt, twitterUser.ID, twitterUser.Username, twitterUser.ProfileImageURL); err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=save_failed")
	}

	// Delete the session (ignore error - session cleanup is best effort)
	_ = h.store.DeleteTwitterPKCESession(ctx, state)

	return c.Redirect(http.StatusFound, "/dashboard?success=twitter_connected")
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
