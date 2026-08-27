package twitter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken_Success(t *testing.T) {
	var gotForm url.Values
	var gotUser, gotPass string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		gotUser, gotPass, _ = r.BasicAuth()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Tokens{
			TokenType:    "bearer",
			ExpiresIn:    7200,
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			Scope:        "tweet.write offline.access",
		})
	}))
	defer server.Close()

	c := NewHTTPClient(WithTokenURL(server.URL), WithCredentials("client-id", "client-secret"))
	tokens, err := c.RefreshToken(context.Background(), "old-refresh")

	require.NoError(t, err)
	assert.Equal(t, "new-access", tokens.AccessToken)
	assert.Equal(t, "new-refresh", tokens.RefreshToken)
	assert.Equal(t, 7200, tokens.ExpiresIn)
	assert.Equal(t, "refresh_token", gotForm.Get("grant_type"))
	assert.Equal(t, "old-refresh", gotForm.Get("refresh_token"))
	assert.Equal(t, "client-id", gotUser)
	assert.Equal(t, "client-secret", gotPass)
}

func TestRefreshToken_KeepsCurrentRefreshTokenWhenNotRotated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Tokens{ExpiresIn: 7200, AccessToken: "new-access"})
	}))
	defer server.Close()

	c := NewHTTPClient(WithTokenURL(server.URL))
	tokens, err := c.RefreshToken(context.Background(), "old-refresh")

	require.NoError(t, err)
	assert.Equal(t, "old-refresh", tokens.RefreshToken)
}

func TestRefreshToken_InvalidGrantRequiresReauth(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		}))

		c := NewHTTPClient(WithTokenURL(server.URL))
		_, err := c.RefreshToken(context.Background(), "revoked")

		assert.ErrorIs(t, err, ErrReauthRequired, "status %d", status)
		server.Close()
	}
}

func TestRefreshToken_ServerErrorIsNotReauth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewHTTPClient(WithTokenURL(server.URL))
	_, err := c.RefreshToken(context.Background(), "still-valid")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrReauthRequired)
	apiErr, ok := IsAPIError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, apiErr.StatusCode)
}

func TestRefreshToken_WithoutRefreshTokenRequiresReauth(t *testing.T) {
	c := NewHTTPClient(WithTokenURL("http://127.0.0.1:0"))
	_, err := c.RefreshToken(context.Background(), "")

	assert.ErrorIs(t, err, ErrReauthRequired)
}

func TestExchangeToken(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		_ = json.NewEncoder(w).Encode(Tokens{ExpiresIn: 7200, AccessToken: "access", RefreshToken: "refresh"})
	}))
	defer server.Close()

	c := NewHTTPClient(WithTokenURL(server.URL))
	tokens, err := c.ExchangeToken(context.Background(), "code", "https://example.com/callback", "verifier")

	require.NoError(t, err)
	assert.Equal(t, "access", tokens.AccessToken)
	assert.Equal(t, "authorization_code", gotForm.Get("grant_type"))
	assert.Equal(t, "code", gotForm.Get("code"))
	assert.Equal(t, "https://example.com/callback", gotForm.Get("redirect_uri"))
	assert.Equal(t, "verifier", gotForm.Get("code_verifier"))
}

func TestPostTweet(t *testing.T) {
	var gotBody map[string]string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	c := NewHTTPClient(WithTweetURL(server.URL))
	err := c.PostTweet(context.Background(), "access-token", "now playing")

	require.NoError(t, err)
	assert.Equal(t, "now playing", gotBody["text"])
	assert.Equal(t, "Bearer access-token", gotAuth)
}

func TestPostTweet_UnauthorizedIsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	c := NewHTTPClient(WithTweetURL(server.URL))
	err := c.PostTweet(context.Background(), "expired", "now playing")

	apiErr, ok := IsAPIError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Contains(t, apiErr.Error(), "Unauthorized")
}

func TestGetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"id":"1","name":"Test","username":"test","profile_image_url":"https://example.com/a.jpg"}}`))
	}))
	defer server.Close()

	c := NewHTTPClient(WithUserURL(server.URL))
	user, err := c.GetUserInfo(context.Background(), "access-token")

	require.NoError(t, err)
	assert.Equal(t, "1", user.ID)
	assert.Equal(t, "test", user.Username)
	assert.Equal(t, "https://example.com/a.jpg", user.ProfileImageURL)
}

func TestIsAPIError_WrappedError(t *testing.T) {
	wrapped := errors.Join(&APIError{StatusCode: 429}, errors.New("other"))
	apiErr, ok := IsAPIError(wrapped)
	require.True(t, ok)
	assert.Equal(t, 429, apiErr.StatusCode)
}
