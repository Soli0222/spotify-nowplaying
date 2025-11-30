package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client はSpotify APIクライアントのインターフェース
type Client interface {
	// GetPlayerData は現在再生中の情報を取得する
	GetPlayerData(accessToken string) (*PlayerResponse, time.Duration, error)
	// ExchangeToken は認証コードをアクセストークンに交換する
	ExchangeToken(code, redirectURI string) (*Tokens, error)
}

// PlayerResponse はSpotify Player APIのレスポンス
type PlayerResponse struct {
	CurrentlyPlayingType string `json:"currently_playing_type"`
	Item                 Item   `json:"item"`
}

// HTTPClient はHTTP通信を行うクライアント
type HTTPClient struct {
	client       *http.Client
	tokenURL     string
	playerURL    string
	clientID     string
	clientSecret string
	logger       *slog.Logger
}

// ClientOption はHTTPClientの設定オプション
type ClientOption func(*HTTPClient)

// WithHTTPClient はカスタムHTTPクライアントを設定する
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *HTTPClient) {
		c.client = client
	}
}

// WithTokenURL はトークンURLを設定する（テスト用）
func WithTokenURL(url string) ClientOption {
	return func(c *HTTPClient) {
		c.tokenURL = url
	}
}

// WithPlayerURL はプレイヤーURLを設定する（テスト用）
func WithPlayerURL(url string) ClientOption {
	return func(c *HTTPClient) {
		c.playerURL = url
	}
}

// WithLogger はロガーを設定する
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *HTTPClient) {
		c.logger = logger
	}
}

// NewHTTPClient は新しいHTTPClientを作成する
func NewHTTPClient(opts ...ClientOption) *HTTPClient {
	c := &HTTPClient{
		client:       &http.Client{Timeout: 10 * time.Second},
		tokenURL:     "https://accounts.spotify.com/api/token",
		playerURL:    "https://api.spotify.com/v1/me/player?market=JP",
		clientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		clientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		logger:       slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// GetPlayerData は現在再生中の情報を取得する
func (c *HTTPClient) GetPlayerData(accessToken string) (*PlayerResponse, time.Duration, error) {
	start := time.Now()

	req, err := http.NewRequest("GET", c.playerURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept-Language", "ja")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return &PlayerResponse{}, duration, &APIError{StatusCode: resp.StatusCode}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, fmt.Errorf("failed to read response: %w", err)
	}

	var playerResp PlayerResponse
	if err := json.Unmarshal(body, &playerResp); err != nil {
		c.logger.Error("failed to unmarshal player response", "error", err)
		return nil, duration, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &playerResp, duration, nil
}

// ExchangeToken は認証コードをアクセストークンに交換する
func (c *HTTPClient) ExchangeToken(code, redirectURI string) (*Tokens, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("token exchange failed", "status", resp.StatusCode, "body", string(body))
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tokens Tokens
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &tokens, nil
}

// APIError はSpotify APIエラーを表す
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("spotify API error: status %d, message: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("spotify API error: status %d", e.StatusCode)
}

// IsAPIError はエラーがAPIErrorかどうかを判定する
func IsAPIError(err error) (*APIError, bool) {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr, true
	}
	return nil, false
}
