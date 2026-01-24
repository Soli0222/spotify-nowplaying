package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/spotify"
	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// APIPostHandler handles API-based posting
type APIPostHandler struct {
	store         *store.Store
	spotifyClient spotify.Client
}

// NewAPIPostHandler creates a new APIPostHandler
func NewAPIPostHandler(s *store.Store, client spotify.Client) *APIPostHandler {
	return &APIPostHandler{
		store:         s,
		spotifyClient: client,
	}
}

// PostTarget represents the target platform for posting
type PostTarget string

const (
	PostTargetMisskey PostTarget = "misskey"
	PostTargetTwitter PostTarget = "twitter"
	PostTargetBoth    PostTarget = "both"
)

// PostResponse represents the response from posting
type PostResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message,omitempty"`
	Results map[string]string `json:"results,omitempty"`
}

// MisskeyNoteRequest represents the request body for creating a Misskey note
type MisskeyNoteRequest struct {
	I          string `json:"i"`
	Text       string `json:"text"`
	Visibility string `json:"visibility,omitempty"`
}

// TwitterTweetRequest represents the request body for creating a Twitter tweet
type TwitterTweetRequest struct {
	Text string `json:"text"`
}

// PostNowPlaying posts the currently playing track to configured platforms
// GET /api/post/:token
func (h *APIPostHandler) PostNowPlaying(c echo.Context) error {
	tokenStr := c.Param("token")
	if tokenStr == "" {
		return c.JSON(http.StatusBadRequest, PostResponse{Success: false, Message: "missing token"})
	}

	apiToken, err := uuid.Parse(tokenStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, PostResponse{Success: false, Message: "invalid token"})
	}

	ctx := c.Request().Context()

	// Get user by API token
	user, err := h.store.GetUserByAPIToken(ctx, apiToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, PostResponse{Success: false, Message: "database error"})
	}
	if user == nil {
		return c.JSON(http.StatusNotFound, PostResponse{Success: false, Message: "token not found"})
	}

	// Check header token if enabled
	if user.APIHeaderTokenEnabled {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, PostResponse{Success: false, Message: "authorization header required"})
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.JSON(http.StatusUnauthorized, PostResponse{Success: false, Message: "invalid authorization header format"})
		}

		providedToken := parts[1]
		providedHash := auth.HashToken(providedToken)

		if !user.APIHeaderTokenHash.Valid || providedHash != user.APIHeaderTokenHash.String {
			return c.JSON(http.StatusUnauthorized, PostResponse{Success: false, Message: "invalid token"})
		}
	}

	// Get target from query param (default: both)
	targetStr := c.QueryParam("target")
	var target PostTarget
	switch strings.ToLower(targetStr) {
	case "misskey":
		target = PostTargetMisskey
	case "twitter":
		target = PostTargetTwitter
	default:
		target = PostTargetBoth
	}

	// Check if Spotify token is available
	if !user.SpotifyAccessToken.Valid || user.SpotifyAccessToken.String == "" {
		return c.JSON(http.StatusBadRequest, PostResponse{Success: false, Message: "spotify not connected"})
	}

	// Get currently playing from Spotify
	accessToken := user.SpotifyAccessToken.String
	playerResp, _, err := h.spotifyClient.GetPlayerData(accessToken)
	if err != nil {
		if apiErr, ok := spotify.IsAPIError(err); ok {
			if apiErr.StatusCode == 401 {
				// Token expired, try to refresh
				if !user.SpotifyRefreshToken.Valid || user.SpotifyRefreshToken.String == "" {
					return c.JSON(http.StatusUnauthorized, PostResponse{Success: false, Message: "spotify token expired and no refresh token available"})
				}

				newTokens, refreshErr := h.spotifyClient.RefreshToken(user.SpotifyRefreshToken.String)
				if refreshErr != nil {
					return c.JSON(http.StatusUnauthorized, PostResponse{Success: false, Message: "failed to refresh spotify token"})
				}

				// Update tokens in database
				expiresAt := time.Now().Add(time.Duration(newTokens.ExpiresIn) * time.Second)
				if updateErr := h.store.UpdateSpotifyToken(ctx, user.ID, newTokens.AccessToken, newTokens.RefreshToken, expiresAt); updateErr != nil {
					return c.JSON(http.StatusInternalServerError, PostResponse{Success: false, Message: "failed to update spotify token"})
				}

				// Retry with new access token
				accessToken = newTokens.AccessToken
				playerResp, _, err = h.spotifyClient.GetPlayerData(accessToken)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, PostResponse{Success: false, Message: "failed to get player data after token refresh"})
				}
			} else {
				return c.JSON(http.StatusBadRequest, PostResponse{Success: false, Message: fmt.Sprintf("spotify api error: %d", apiErr.StatusCode)})
			}
		} else {
			return c.JSON(http.StatusInternalServerError, PostResponse{Success: false, Message: "failed to get player data"})
		}
	}

	// Parse player response to get track data
	trackData, contentType := spotify.ParsePlayerResponse(playerResp)
	if contentType == "unknown" {
		return c.JSON(http.StatusOK, PostResponse{Success: false, Message: "nothing is playing"})
	}

	// Build the post text (same format as existing implementation)
	var postText string
	switch contentType {
	case "track":
		postText = fmt.Sprintf("%s / %s\n#NowPlaying #PsrPlaying\n%s", trackData.TrackName, trackData.ArtistName, trackData.TrackURL)
	case "episode":
		postText = fmt.Sprintf("%s / %s\n#NowPlaying\n%s", trackData.TrackName, trackData.ArtistName, trackData.TrackURL)
	}

	results := make(map[string]string)

	// Post to Misskey
	if target == PostTargetMisskey || target == PostTargetBoth {
		if user.MisskeyAccessToken.Valid && user.MisskeyAccessToken.String != "" {
			err := h.postToMisskey(user.MisskeyInstanceURL.String, user.MisskeyAccessToken.String, postText)
			if err != nil {
				results["misskey"] = fmt.Sprintf("error: %s", err.Error())
			} else {
				results["misskey"] = "success"
			}
		} else {
			results["misskey"] = "not connected"
		}
	}

	// Post to Twitter
	if target == PostTargetTwitter || target == PostTargetBoth {
		if user.TwitterAccessToken.Valid && user.TwitterAccessToken.String != "" {
			err := h.postToTwitter(user.TwitterAccessToken.String, postText)
			if err != nil {
				results["twitter"] = fmt.Sprintf("error: %s", err.Error())
			} else {
				results["twitter"] = "success"
			}
		} else {
			results["twitter"] = "not connected"
		}
	}

	// Check if any succeeded
	anySuccess := false
	for _, v := range results {
		if v == "success" {
			anySuccess = true
			break
		}
	}

	return c.JSON(http.StatusOK, PostResponse{
		Success: anySuccess,
		Message: postText,
		Results: results,
	})
}

// postToMisskey posts a note to Misskey
func (h *APIPostHandler) postToMisskey(instanceURL, accessToken, text string) error {
	// Ensure instance URL has protocol
	if !strings.HasPrefix(instanceURL, "http://") && !strings.HasPrefix(instanceURL, "https://") {
		instanceURL = "https://" + instanceURL
	}

	reqBody := MisskeyNoteRequest{
		I:          accessToken,
		Text:       text,
		Visibility: "public",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/notes/create", instanceURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("misskey api error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// postToTwitter posts a tweet to Twitter
func (h *APIPostHandler) postToTwitter(accessToken, text string) error {
	reqBody := TwitterTweetRequest{
		Text: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.twitter.com/2/tweets", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitter api error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}
