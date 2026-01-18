package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// MiAuthHandler handles MiAuth authentication
type MiAuthHandler struct {
	store     *store.Store
	jwtConfig auth.JWTConfig
}

// NewMiAuthHandler creates a new MiAuthHandler
func NewMiAuthHandler(s *store.Store, jwtConfig auth.JWTConfig) *MiAuthHandler {
	return &MiAuthHandler{
		store:     s,
		jwtConfig: jwtConfig,
	}
}

// MiAuthStartRequest is the request body for starting MiAuth
type MiAuthStartRequest struct {
	InstanceURL string `json:"instance_url"`
}

// MiAuthStartResponse is the response for starting MiAuth
type MiAuthStartResponse struct {
	AuthURL string `json:"auth_url"`
}

// MiAuthCheckResponse is the response from MiAuth check API
type MiAuthCheckResponse struct {
	OK    bool   `json:"ok"`
	Token string `json:"token"`
	User  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

// StartMiAuth starts the MiAuth authentication flow
// POST /api/miauth/start
func (h *MiAuthHandler) StartMiAuth(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var req MiAuthStartRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validate and normalize instance URL
	instanceURL := strings.TrimSpace(req.InstanceURL)
	if instanceURL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance_url is required"})
	}

	// Add https:// if not present
	if !strings.HasPrefix(instanceURL, "http://") && !strings.HasPrefix(instanceURL, "https://") {
		instanceURL = "https://" + instanceURL
	}

	// Remove trailing slash
	instanceURL = strings.TrimSuffix(instanceURL, "/")

	// Generate session ID
	sessionID := uuid.New()

	// Store the session
	ctx := c.Request().Context()
	if err := h.store.CreateMiAuthSession(ctx, userID, sessionID, instanceURL); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
	}

	// Build MiAuth URL
	// https://misskey-hub.net/ja/docs/for-developers/api/token/miauth/
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "Spotify NowPlaying"
	}
	callbackURL := os.Getenv("BASE_URL") + "/api/miauth/callback"

	// Permissions: write:notes (to post notes), read:account (to get user info)
	permission := "write:notes,read:account"

	authURL := fmt.Sprintf("%s/miauth/%s?name=%s&callback=%s&permission=%s",
		instanceURL,
		sessionID.String(),
		url.QueryEscape(appName),
		url.QueryEscape(callbackURL),
		permission,
	)

	return c.JSON(http.StatusOK, MiAuthStartResponse{AuthURL: authURL})
}

// CallbackMiAuth handles the MiAuth callback
// GET /api/miauth/callback
func (h *MiAuthHandler) CallbackMiAuth(c echo.Context) error {
	sessionIDStr := c.QueryParam("session")
	if sessionIDStr == "" {
		return c.Redirect(http.StatusFound, "/dashboard?error=missing_session")
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=invalid_session")
	}

	ctx := c.Request().Context()

	// Get the session
	session, err := h.store.GetMiAuthSession(ctx, sessionID)
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=session_error")
	}
	if session == nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=session_not_found")
	}

	// Call MiAuth check API
	checkURL := fmt.Sprintf("%s/api/miauth/%s/check", session.InstanceURL, sessionID.String())

	resp, err := http.Post(checkURL, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=check_failed")
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=read_failed")
	}

	var checkResp MiAuthCheckResponse
	if err := json.Unmarshal(body, &checkResp); err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=parse_failed")
	}

	if !checkResp.OK || checkResp.Token == "" {
		return c.Redirect(http.StatusFound, "/dashboard?error=auth_failed")
	}

	// Fetch Misskey user info (including avatar)
	misskeyUser, err := h.getMisskeyUserInfo(session.InstanceURL, checkResp.Token)
	if err != nil {
		// If we can't get full user info, use what we have from checkResp
		misskeyUser = &MisskeyUserInfo{
			ID:       checkResp.User.ID,
			Username: checkResp.User.Username,
		}
	} else {
		fmt.Printf("Successfully fetched Misskey user info: ID=%s, Username=%s, AvatarURL=%s\n",
			misskeyUser.ID, misskeyUser.Username, misskeyUser.AvatarURL)
	}

	// Extract host from instance URL
	host := session.InstanceURL
	if parsedURL, err := url.Parse(session.InstanceURL); err == nil {
		host = parsedURL.Host
	}

	// Save the token and user info to user
	if err := h.store.UpdateMisskeyToken(ctx, session.UserID, session.InstanceURL, checkResp.Token, misskeyUser.ID, misskeyUser.Username, misskeyUser.AvatarURL, host); err != nil {
		return c.Redirect(http.StatusFound, "/dashboard?error=save_failed")
	}

	// Delete the session (ignore error - session cleanup is best effort)
	_ = h.store.DeleteMiAuthSession(ctx, sessionID)

	return c.Redirect(http.StatusFound, "/dashboard?success=misskey_connected")
}

// DisconnectMisskey disconnects Misskey from the user account
// DELETE /api/miauth
func (h *MiAuthHandler) DisconnectMisskey(c echo.Context) error {
	userID, err := auth.GetUserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Request().Context()
	if err := h.store.DisconnectMisskey(ctx, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to disconnect"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "misskey disconnected"})
}

// MisskeyUserInfo represents Misskey user information
type MisskeyUserInfo struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatarUrl"`
}

// getMisskeyUserInfo fetches the Misskey user information
func (h *MiAuthHandler) getMisskeyUserInfo(instanceURL, accessToken string) (*MisskeyUserInfo, error) {
	reqBody := map[string]string{"i": accessToken}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", instanceURL+"/api/i", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("misskey api error: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo MisskeyUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}
