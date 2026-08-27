// Package twitter provides a small client for the parts of the X (Twitter) API
// this application uses: the OAuth 2.0 PKCE token endpoint, users/me and tweet
// creation.
package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Tokens is the response from the Twitter token endpoint.
type Tokens struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// UserInfo represents Twitter user information.
type UserInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

type userResponse struct {
	Data UserInfo `json:"data"`
}

// ErrReauthRequired is returned when the stored refresh token can no longer be
// used, so the user has to connect their Twitter account again.
var ErrReauthRequired = errors.New("twitter reauthorization required")

// Client is the interface of the Twitter API client.
type Client interface {
	// ExchangeToken exchanges an authorization code for tokens.
	ExchangeToken(ctx context.Context, code, redirectURI, codeVerifier string) (*Tokens, error)
	// RefreshToken exchanges a refresh token for a new set of tokens.
	RefreshToken(ctx context.Context, refreshToken string) (*Tokens, error)
	// GetUserInfo fetches the profile of the authenticated user.
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
	// PostTweet posts a tweet as the authenticated user.
	PostTweet(ctx context.Context, accessToken, text string) error
}

// HTTPClient talks to the real Twitter API.
type HTTPClient struct {
	client       *http.Client
	tokenURL     string
	tweetURL     string
	userURL      string
	clientID     string
	clientSecret string
	logger       *slog.Logger
}

// ClientOption configures an HTTPClient.
type ClientOption func(*HTTPClient)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *HTTPClient) { c.client = client }
}

// WithTokenURL overrides the token endpoint (for tests).
func WithTokenURL(u string) ClientOption {
	return func(c *HTTPClient) { c.tokenURL = u }
}

// WithTweetURL overrides the tweet endpoint (for tests).
func WithTweetURL(u string) ClientOption {
	return func(c *HTTPClient) { c.tweetURL = u }
}

// WithUserURL overrides the users/me endpoint (for tests).
func WithUserURL(u string) ClientOption {
	return func(c *HTTPClient) { c.userURL = u }
}

// WithCredentials overrides the OAuth client credentials (for tests).
func WithCredentials(clientID, clientSecret string) ClientOption {
	return func(c *HTTPClient) {
		c.clientID = clientID
		c.clientSecret = clientSecret
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *HTTPClient) { c.logger = logger }
}

// NewHTTPClient creates a new HTTPClient.
func NewHTTPClient(opts ...ClientOption) *HTTPClient {
	c := &HTTPClient{
		client:       &http.Client{Timeout: 30 * time.Second},
		tokenURL:     "https://api.twitter.com/2/oauth2/token",
		tweetURL:     "https://api.twitter.com/2/tweets",
		userURL:      "https://api.twitter.com/2/users/me?user.fields=profile_image_url",
		clientID:     os.Getenv("TWITTER_CLIENT_ID"),
		clientSecret: os.Getenv("TWITTER_CLIENT_SECRET"),
		logger:       slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ExchangeToken exchanges an authorization code for tokens.
func (c *HTTPClient) ExchangeToken(ctx context.Context, code, redirectURI, codeVerifier string) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", codeVerifier)

	return c.requestTokens(ctx, form, "token exchange")
}

// RefreshToken exchanges a refresh token for a new set of tokens.
// Twitter rotates the refresh token, so the returned one must be persisted.
func (c *HTTPClient) RefreshToken(ctx context.Context, refreshToken string) (*Tokens, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("%w: no refresh token stored", ErrReauthRequired)
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)

	tokens, err := c.requestTokens(ctx, form, "token refresh")
	if err != nil {
		return nil, err
	}

	// Defensive: keep the current refresh token if Twitter did not rotate it.
	if tokens.RefreshToken == "" {
		tokens.RefreshToken = refreshToken
	}

	return tokens, nil
}

func (c *HTTPClient) requestTokens(ctx context.Context, form url.Values, operation string) (*Tokens, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error(operation+" failed", "status", resp.StatusCode, "body", string(body))
		apiErr := &APIError{StatusCode: resp.StatusCode, Message: string(body)}
		// The grant itself was rejected: refreshing again will not help.
		if resp.StatusCode == http.StatusBadRequest ||
			resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("%w: %v", ErrReauthRequired, apiErr)
		}
		return nil, apiErr
	}

	var tokens Tokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &tokens, nil
}

// GetUserInfo fetches the profile of the authenticated user.
func (c *HTTPClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	var userResp userResponse
	if err := json.Unmarshal(body, &userResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &userResp.Data, nil
}

// PostTweet posts a tweet as the authenticated user.
func (c *HTTPClient) PostTweet(ctx context.Context, accessToken, text string) error {
	jsonBody, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tweetURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	return nil
}

// APIError represents a Twitter API error.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("twitter api error: %d - %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("twitter api error: %d", e.StatusCode)
}

// IsAPIError reports whether err is an APIError.
func IsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
