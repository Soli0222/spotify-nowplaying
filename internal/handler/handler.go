package handler

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/Soli0222/spotify-nowplaying/internal/metrics"
	"github.com/Soli0222/spotify-nowplaying/internal/spotify"

	"github.com/labstack/echo/v4"
)

// Platform はシェア先プラットフォームを表す
type Platform string

const (
	PlatformMisskey Platform = "Misskey"
	PlatformTwitter Platform = "Twitter"
)

// Handler はHTTPハンドラーを管理する構造体
type Handler struct {
	spotifyClient spotify.Client
}

// NewHandler は新しいHandlerを作成する
func NewHandler(client spotify.Client) *Handler {
	return &Handler{
		spotifyClient: client,
	}
}

// homeHandler は共通のホームハンドラー処理
func (h *Handler) homeHandler(c echo.Context, platform Platform, platformLabel string) error {
	cookie, err := c.Cookie("access_token")
	if err != nil {
		return c.String(http.StatusUnauthorized, "Access token not found. Please login first.")
	}

	accessToken := cookie.Value

	playerResp, duration, err := h.spotifyClient.GetPlayerData(accessToken)
	if err != nil {
		if apiErr, ok := spotify.IsAPIError(err); ok {
			metrics.SpotifyAPIRequestDuration.WithLabelValues("player").Observe(duration.Seconds())
			metrics.SpotifyAPIRequestsTotal.WithLabelValues("player", strconv.Itoa(apiErr.StatusCode)).Inc()
			statusText := http.StatusText(apiErr.StatusCode)
			return c.String(http.StatusOK, fmt.Sprintf("%d: %s", apiErr.StatusCode, statusText))
		}
		metrics.SpotifyAPIRequestsTotal.WithLabelValues("player", "error").Inc()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	metrics.SpotifyAPIRequestDuration.WithLabelValues("player").Observe(duration.Seconds())
	metrics.SpotifyAPIRequestsTotal.WithLabelValues("player", "200").Inc()

	shareURL, contentType := spotify.GetShareInfo(playerResp, string(platform))

	metrics.ShareRedirectsTotal.WithLabelValues(platformLabel, contentType).Inc()
	return c.Redirect(http.StatusFound, shareURL)
}

// loginHandler は共通のログインハンドラー処理
func loginHandler(c echo.Context, callbackPath string) error {
	authURL := "https://accounts.spotify.com/authorize"
	scope := "user-read-currently-playing user-read-playback-state"
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	baseURL := os.Getenv("BASE_URL")
	redirectURI := baseURL + callbackPath

	redirectURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s",
		authURL, clientID, redirectURI, scope)

	return c.Redirect(http.StatusFound, redirectURL)
}

// callbackHandler は共通のコールバックハンドラー処理
func (h *Handler) callbackHandler(c echo.Context, platform Platform, platformLabel, homePath string) error {
	code := c.QueryParam("code")

	if code == "" {
		metrics.OAuthCallbacksTotal.WithLabelValues(platformLabel, "missing_code").Inc()
		return c.String(http.StatusBadRequest, "Code parameter is missing.")
	}

	baseURL := os.Getenv("BASE_URL")
	var redirectURI string
	switch platform {
	case PlatformMisskey:
		redirectURI = baseURL + "/note/callback"
	case PlatformTwitter:
		redirectURI = baseURL + "/tweet/callback"
	}

	tokens, err := h.spotifyClient.ExchangeToken(code, redirectURI)
	if err != nil {
		metrics.OAuthCallbacksTotal.WithLabelValues(platformLabel, "error").Inc()
		return c.String(http.StatusInternalServerError, "OAuth callback failed")
	}

	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    tokens.AccessToken,
		HttpOnly: true,
	}
	c.SetCookie(cookie)

	metrics.OAuthCallbacksTotal.WithLabelValues(platformLabel, "success").Inc()
	return c.Redirect(http.StatusFound, homePath)
}
