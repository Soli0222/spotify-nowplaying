package spotify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_GetPlayerData_Success(t *testing.T) {
	// モックサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "ja", r.Header.Get("Accept-Language"))

		response := PlayerResponse{
			CurrentlyPlayingType: "track",
			Item: Item{
				Name:    "Test Song",
				Artists: []Artist{{Name: "Test Artist"}},
				ExternalUrls: ExternalUrls{
					Spotify: "https://open.spotify.com/track/123",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithPlayerURL(server.URL),
	)

	resp, duration, err := client.GetPlayerData("test-token")

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "track", resp.CurrentlyPlayingType)
	assert.Equal(t, "Test Song", resp.Item.Name)
	assert.Greater(t, duration.Nanoseconds(), int64(0))
}

func TestHTTPClient_GetPlayerData_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithPlayerURL(server.URL),
	)

	resp, duration, err := client.GetPlayerData("invalid-token")

	require.Error(t, err)
	assert.NotNil(t, resp)
	assert.Greater(t, duration.Nanoseconds(), int64(0))

	apiErr, ok := IsAPIError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
}

func TestHTTPClient_GetPlayerData_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithPlayerURL(server.URL),
	)

	resp, _, err := client.GetPlayerData("test-token")

	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestHTTPClient_ExchangeToken_Success(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "test-client-id")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "test-client-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("Authorization"), "Basic")
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "test-code", r.Form.Get("code"))
		assert.Equal(t, "http://localhost/callback", r.Form.Get("redirect_uri"))

		response := Tokens{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithTokenURL(server.URL),
	)

	tokens, err := client.ExchangeToken("test-code", "http://localhost/callback")

	require.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.Equal(t, "new-access-token", tokens.AccessToken)
	assert.Equal(t, "new-refresh-token", tokens.RefreshToken)
}

func TestHTTPClient_ExchangeToken_APIError(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "test-client-id")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "test-client-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(
		WithTokenURL(server.URL),
	)

	tokens, err := client.ExchangeToken("invalid-code", "http://localhost/callback")

	require.Error(t, err)
	assert.Nil(t, tokens)

	apiErr, ok := IsAPIError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 401, Message: "Unauthorized"}
	assert.Contains(t, err.Error(), "401")
	assert.Contains(t, err.Error(), "Unauthorized")

	err2 := &APIError{StatusCode: 500}
	assert.Contains(t, err2.Error(), "500")
}

func TestIsAPIError(t *testing.T) {
	apiErr := &APIError{StatusCode: 401}
	result, ok := IsAPIError(apiErr)
	assert.True(t, ok)
	assert.Equal(t, apiErr, result)

	otherErr := assert.AnError
	result2, ok2 := IsAPIError(otherErr)
	assert.False(t, ok2)
	assert.Nil(t, result2)
}

func TestNewHTTPClient_Defaults(t *testing.T) {
	t.Setenv("SPOTIFY_CLIENT_ID", "env-client-id")
	t.Setenv("SPOTIFY_CLIENT_SECRET", "env-client-secret")

	client := NewHTTPClient()

	assert.NotNil(t, client.client)
	assert.Equal(t, "https://accounts.spotify.com/api/token", client.tokenURL)
	assert.Equal(t, "https://api.spotify.com/v1/me/player?market=JP", client.playerURL)
	assert.Equal(t, "env-client-id", client.clientID)
	assert.Equal(t, "env-client-secret", client.clientSecret)
}

func TestNewHTTPClient_WithOptions(t *testing.T) {
	customClient := &http.Client{}

	client := NewHTTPClient(
		WithHTTPClient(customClient),
		WithTokenURL("https://custom.token.url"),
		WithPlayerURL("https://custom.player.url"),
	)

	assert.Equal(t, customClient, client.client)
	assert.Equal(t, "https://custom.token.url", client.tokenURL)
	assert.Equal(t, "https://custom.player.url", client.playerURL)
}
