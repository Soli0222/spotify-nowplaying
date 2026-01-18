package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpotifyAuthHandler_LoginSpotify(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "test-client-id")
	t.Setenv("BASE_URL", "http://localhost:8080")

	handler := NewSpotifyAuthHandler(nil, nil, auth.JWTConfig{})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/spotify", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.LoginSpotify(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)

	location := rec.Header().Get("Location")
	assert.Contains(t, location, "https://accounts.spotify.com/authorize")
	assert.Contains(t, location, "client_id=test-client-id")
	assert.Contains(t, location, "redirect_uri=http://localhost:8080/api/auth/spotify/callback")
	assert.Contains(t, location, "user-read-currently-playing")
	assert.Contains(t, location, "user-read-playback-state")
}

func TestSpotifyAuthHandler_CheckAuth_NotAuthenticated(t *testing.T) {
	jwtConfig := auth.JWTConfig{
		SecretKey:  "test-secret",
		CookieName: "session_token",
	}
	handler := NewSpotifyAuthHandler(nil, nil, jwtConfig)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.CheckAuth(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, false, response["authenticated"])
}

func TestSpotifyAuthHandler_CheckAuth_Authenticated(t *testing.T) {
	jwtConfig := auth.JWTConfig{
		SecretKey:     "test-secret",
		CookieName:    "session_token",
		TokenDuration: time.Hour,
	}

	userID := uuid.New()
	token, err := auth.GenerateToken(jwtConfig, userID, "spotify-user-123")
	require.NoError(t, err)

	handler := NewSpotifyAuthHandler(nil, nil, jwtConfig)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.CheckAuth(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["authenticated"])
	assert.Equal(t, userID.String(), response["user_id"])
	assert.Equal(t, "spotify-user-123", response["spotify_user_id"])
}

func TestSpotifyAuthHandler_CheckAuth_InvalidToken(t *testing.T) {
	jwtConfig := auth.JWTConfig{
		SecretKey:  "test-secret",
		CookieName: "session_token",
	}

	handler := NewSpotifyAuthHandler(nil, nil, jwtConfig)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: "invalid-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.CheckAuth(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, false, response["authenticated"])
}

func TestSpotifyUserResponse_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": "user123",
		"display_name": "Test User",
		"email": "test@example.com",
		"images": [
			{"url": "https://example.com/image1.jpg", "height": 640, "width": 640},
			{"url": "https://example.com/image2.jpg", "height": 320, "width": 320}
		]
	}`

	var resp SpotifyUserResponse
	err := json.Unmarshal([]byte(jsonData), &resp)

	require.NoError(t, err)
	assert.Equal(t, "user123", resp.ID)
	assert.Equal(t, "Test User", resp.DisplayName)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Len(t, resp.Images, 2)
	assert.Equal(t, "https://example.com/image1.jpg", resp.Images[0].URL)
}

func TestSpotifyImage_JSONUnmarshal(t *testing.T) {
	jsonData := `{"url": "https://example.com/image.jpg", "height": 640, "width": 640}`

	var img SpotifyImage
	err := json.Unmarshal([]byte(jsonData), &img)

	require.NoError(t, err)
	assert.Equal(t, "https://example.com/image.jpg", img.URL)
	assert.Equal(t, 640, img.Height)
	assert.Equal(t, 640, img.Width)
}
