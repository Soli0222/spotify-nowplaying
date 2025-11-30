package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/spotify"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSpotifyClient はテスト用のモッククライアント
type MockSpotifyClient struct {
	GetPlayerDataFunc func(accessToken string) (*spotify.PlayerResponse, time.Duration, error)
	ExchangeTokenFunc func(code, redirectURI string) (*spotify.Tokens, error)
}

func (m *MockSpotifyClient) GetPlayerData(accessToken string) (*spotify.PlayerResponse, time.Duration, error) {
	if m.GetPlayerDataFunc != nil {
		return m.GetPlayerDataFunc(accessToken)
	}
	return nil, 0, errors.New("not implemented")
}

func (m *MockSpotifyClient) ExchangeToken(code, redirectURI string) (*spotify.Tokens, error) {
	if m.ExchangeTokenFunc != nil {
		return m.ExchangeTokenFunc(code, redirectURI)
	}
	return nil, errors.New("not implemented")
}

func TestStatusHandler(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := StatusHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(http.StatusOK), response["status_code"])
	assert.Contains(t, response, "response_time")
}

func TestNoteLoginHandler(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "test-client-id")
	t.Setenv("SPOTIFY_REDIRECT_URI_NOTE", "http://localhost/note/callback")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/note", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := NoteLoginHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)

	location := rec.Header().Get("Location")
	assert.Contains(t, location, "https://accounts.spotify.com/authorize")
	assert.Contains(t, location, "client_id=test-client-id")
	assert.Contains(t, location, "redirect_uri=http://localhost/note/callback")
	assert.Contains(t, location, "scope=user-read-currently-playing")
}

func TestTweetLoginHandler(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "test-client-id")
	t.Setenv("SPOTIFY_REDIRECT_URI_TWEET", "http://localhost/tweet/callback")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tweet", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := TweetLoginHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)

	location := rec.Header().Get("Location")
	assert.Contains(t, location, "https://accounts.spotify.com/authorize")
	assert.Contains(t, location, "client_id=test-client-id")
	assert.Contains(t, location, "redirect_uri=http://localhost/tweet/callback")
}

func TestNoteCallbackHandler_MissingCode(t *testing.T) {
	mockClient := &MockSpotifyClient{}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/note/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.NoteCallbackHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Code parameter is missing")
}

func TestTweetCallbackHandler_MissingCode(t *testing.T) {
	mockClient := &MockSpotifyClient{}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tweet/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TweetCallbackHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Code parameter is missing")
}

func TestNoteCallbackHandler_Success(t *testing.T) {
	t.Setenv("SPOTIFY_REDIRECT_URI_NOTE", "http://localhost/note/callback")

	mockClient := &MockSpotifyClient{
		ExchangeTokenFunc: func(code, redirectURI string) (*spotify.Tokens, error) {
			assert.Equal(t, "test-code", code)
			assert.Equal(t, "http://localhost/note/callback", redirectURI)
			return &spotify.Tokens{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
			}, nil
		},
	}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/note/callback?code=test-code", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.NoteCallbackHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/note/home", rec.Header().Get("Location"))

	cookies := rec.Result().Cookies()
	var accessTokenCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "access_token" {
			accessTokenCookie = cookie
			break
		}
	}
	require.NotNil(t, accessTokenCookie)
	assert.Equal(t, "test-access-token", accessTokenCookie.Value)
}

func TestTweetCallbackHandler_Success(t *testing.T) {
	t.Setenv("SPOTIFY_REDIRECT_URI_TWEET", "http://localhost/tweet/callback")

	mockClient := &MockSpotifyClient{
		ExchangeTokenFunc: func(code, redirectURI string) (*spotify.Tokens, error) {
			assert.Equal(t, "test-code", code)
			return &spotify.Tokens{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
			}, nil
		},
	}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tweet/callback?code=test-code", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TweetCallbackHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/tweet/home", rec.Header().Get("Location"))
}

func TestNoteHomeHandler_NoCookie(t *testing.T) {
	mockClient := &MockSpotifyClient{}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/note/home", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.NoteHomeHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Access token not found")
}

func TestTweetHomeHandler_NoCookie(t *testing.T) {
	mockClient := &MockSpotifyClient{}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tweet/home", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TweetHomeHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "Access token not found")
}

func TestNoteHomeHandler_Success(t *testing.T) {
	t.Setenv("SERVER_URI", "misskey.io")

	mockClient := &MockSpotifyClient{
		GetPlayerDataFunc: func(accessToken string) (*spotify.PlayerResponse, time.Duration, error) {
			assert.Equal(t, "test-access-token", accessToken)
			return &spotify.PlayerResponse{
				CurrentlyPlayingType: "track",
				Item: spotify.Item{
					Name:    "Test Song",
					Artists: []spotify.Artist{{Name: "Test Artist"}},
					ExternalUrls: spotify.ExternalUrls{
						Spotify: "https://open.spotify.com/track/123",
					},
				},
			}, 100 * time.Millisecond, nil
		},
	}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/note/home", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-access-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.NoteHomeHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "misskey.io/share")
}

func TestTweetHomeHandler_Success(t *testing.T) {
	mockClient := &MockSpotifyClient{
		GetPlayerDataFunc: func(accessToken string) (*spotify.PlayerResponse, time.Duration, error) {
			return &spotify.PlayerResponse{
				CurrentlyPlayingType: "track",
				Item: spotify.Item{
					Name:    "Test Song",
					Artists: []spotify.Artist{{Name: "Test Artist"}},
					ExternalUrls: spotify.ExternalUrls{
						Spotify: "https://open.spotify.com/track/123",
					},
				},
			}, 100 * time.Millisecond, nil
		},
	}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/tweet/home", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-access-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.TweetHomeHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get("Location")
	assert.Contains(t, location, "x.com/intent/tweet")
}

func TestHomeHandler_APIError(t *testing.T) {
	mockClient := &MockSpotifyClient{
		GetPlayerDataFunc: func(accessToken string) (*spotify.PlayerResponse, time.Duration, error) {
			return &spotify.PlayerResponse{}, 50 * time.Millisecond, &spotify.APIError{StatusCode: 401}
		},
	}
	h := NewHandler(mockClient)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/note/home", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: "test-access-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.NoteHomeHandler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "401")
}

func TestPlatformConstants(t *testing.T) {
	assert.Equal(t, Platform("Misskey"), PlatformMisskey)
	assert.Equal(t, Platform("Twitter"), PlatformTwitter)
}
