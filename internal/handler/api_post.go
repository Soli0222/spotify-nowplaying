package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/spotify"
	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/Soli0222/spotify-nowplaying/internal/twitter"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// APIPostHandler handles API-based posting
type APIPostHandler struct {
	store         *store.Store
	spotifyClient spotify.Client
	twitterClient twitter.Client
}

// NewAPIPostHandler creates a new APIPostHandler
func NewAPIPostHandler(s *store.Store, client spotify.Client, twitterClient twitter.Client) *APIPostHandler {
	return &APIPostHandler{
		store:         s,
		spotifyClient: client,
		twitterClient: twitterClient,
	}
}

// twitterTokenLeeway is how long before the recorded expiry the Twitter access
// token is refreshed. Twitter access tokens are valid for two hours.
const twitterTokenLeeway = 5 * time.Minute

// twitterTokenNeedsRefresh reports whether the access token expiring at
// expiresAt should be refreshed before use. An unknown expiry (zero value, e.g.
// a row written before expiries were stored) is treated as expired.
func twitterTokenNeedsRefresh(expiresAt time.Time, now time.Time) bool {
	if expiresAt.IsZero() {
		return true
	}
	return !now.Add(twitterTokenLeeway).Before(expiresAt)
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

	if !user.APIHeaderTokenEnabled || !user.APIHeaderTokenHash.Valid || user.APIHeaderTokenHash.String == "" {
		return c.JSON(http.StatusUnauthorized, PostResponse{Success: false, Message: "header token is required"})
	}

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

	if providedHash != user.APIHeaderTokenHash.String {
		return c.JSON(http.StatusUnauthorized, PostResponse{Success: false, Message: "invalid token"})
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
	attempted := 0
	succeeded := 0

	// Post to Misskey
	if target == PostTargetMisskey || target == PostTargetBoth {
		if user.MisskeyAccessToken.Valid && user.MisskeyAccessToken.String != "" {
			attempted++
			err := h.postToMisskey(user.MisskeyInstanceURL.String, user.MisskeyAccessToken.String, postText)
			if err != nil {
				results["misskey"] = fmt.Sprintf("error: %s", err.Error())
			} else {
				results["misskey"] = "success"
				succeeded++
			}
		} else {
			results["misskey"] = "not connected"
		}
	}

	// Post to Twitter
	if target == PostTargetTwitter || target == PostTargetBoth {
		if user.TwitterAccessToken.Valid && user.TwitterAccessToken.String != "" {
			attempted++
			err := h.postToTwitter(ctx, user.ID, postText)
			switch {
			case err == nil:
				results["twitter"] = "success"
				succeeded++
			case errors.Is(err, twitter.ErrReauthRequired):
				results["twitter"] = "error: twitter token expired, reconnect required"
			case errors.Is(err, store.ErrTwitterNotConnected):
				results["twitter"] = "not connected"
			default:
				results["twitter"] = fmt.Sprintf("error: %s", err.Error())
			}
		} else {
			results["twitter"] = "not connected"
		}
	}

	// Every platform we actually tried to post to failed: report it as an error
	// so callers and monitoring do not read a 200 as a successful post.
	status := http.StatusOK
	if attempted > 0 && succeeded == 0 {
		status = http.StatusBadGateway
	}

	return c.JSON(status, PostResponse{
		Success: succeeded > 0,
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
	var err error
	instanceURL, err = validatePublicHTTPSURL(strings.TrimSuffix(instanceURL, "/"))
	if err != nil {
		return fmt.Errorf("invalid misskey instance URL: %w", err)
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

// twitterTokenStore is the part of the store postTweetWithRefresh needs.
// *store.Store implements it.
type twitterTokenStore interface {
	EnsureTwitterToken(ctx context.Context, userID uuid.UUID, refresh func(context.Context, store.TwitterTokens) (*store.TwitterTokens, error)) (store.TwitterTokens, error)
}

// twitterPostTimeout bounds a single post, including up to two refreshes.
const twitterPostTimeout = 60 * time.Second

// postToTwitter posts a tweet as the given user.
func (h *APIPostHandler) postToTwitter(ctx context.Context, userID uuid.UUID, text string) error {
	// Detach from the request: if the caller hangs up mid-refresh, the rotated
	// refresh token must still be written, otherwise the account is left with a
	// refresh token Twitter has already invalidated.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), twitterPostTimeout)
	defer cancel()

	return postTweetWithRefresh(ctx, h.store, h.twitterClient, userID, text)
}

// postTweetWithRefresh posts a tweet, refreshing the stored access token before
// posting when it is expired (or about to be), and once more if Twitter answers
// 401 anyway. Refreshes run under the store's per-user lock, so concurrent posts
// cannot invalidate each other's rotated refresh token.
func postTweetWithRefresh(ctx context.Context, tokenStore twitterTokenStore, client twitter.Client, userID uuid.UUID, text string) error {
	tokens, err := tokenStore.EnsureTwitterToken(ctx, userID, func(ctx context.Context, current store.TwitterTokens) (*store.TwitterTokens, error) {
		if !twitterTokenNeedsRefresh(current.ExpiresAt, time.Now()) {
			return nil, nil
		}
		return refreshTwitterTokens(ctx, client, current.RefreshToken)
	})
	if err != nil {
		return err
	}

	postErr := client.PostTweet(ctx, tokens.AccessToken, text)
	if apiErr, ok := twitter.IsAPIError(postErr); !ok || apiErr.StatusCode != http.StatusUnauthorized {
		return postErr
	}

	// The token was rejected even though it still looked valid (clock skew, a
	// token revoked on Twitter's side, ...). Refresh once and retry.
	rejected := tokens.AccessToken
	tokens, err = tokenStore.EnsureTwitterToken(ctx, userID, func(ctx context.Context, current store.TwitterTokens) (*store.TwitterTokens, error) {
		if current.AccessToken != rejected {
			// A concurrent post already refreshed it; use what it stored.
			return nil, nil
		}
		return refreshTwitterTokens(ctx, client, current.RefreshToken)
	})
	if err != nil {
		return err
	}

	return client.PostTweet(ctx, tokens.AccessToken, text)
}

// refreshTwitterTokens exchanges the refresh token for a new set of tokens.
// Twitter rotates the refresh token, so all three values have to be stored.
func refreshTwitterTokens(ctx context.Context, client twitter.Client, refreshToken string) (*store.TwitterTokens, error) {
	tokens, err := client.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	return &store.TwitterTokens{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
	}, nil
}
