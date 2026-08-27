package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/store"
	"github.com/Soli0222/spotify-nowplaying/internal/twitter"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTwitterTokenStore keeps the tokens in memory, mimicking the locked
// read-modify-write the real store performs.
type fakeTwitterTokenStore struct {
	tokens store.TwitterTokens
	// beforeRefresh runs while the "lock" is held, to simulate another request
	// having refreshed the tokens in the meantime.
	beforeRefresh func(s *fakeTwitterTokenStore)
	writes        int
	err           error
}

func (s *fakeTwitterTokenStore) EnsureTwitterToken(ctx context.Context, userID uuid.UUID, refresh func(context.Context, store.TwitterTokens) (*store.TwitterTokens, error)) (store.TwitterTokens, error) {
	if s.err != nil {
		return store.TwitterTokens{}, s.err
	}
	if s.beforeRefresh != nil {
		s.beforeRefresh(s)
	}

	updated, err := refresh(ctx, s.tokens)
	if err != nil {
		return store.TwitterTokens{}, err
	}
	if updated != nil {
		s.tokens = *updated
		s.writes++
	}
	return s.tokens, nil
}

type fakeTwitterClient struct {
	postedTokens  []string
	postErrs      []error
	refreshCalls  []string
	refreshTokens *twitter.Tokens
	refreshErr    error
}

func (c *fakeTwitterClient) ExchangeToken(ctx context.Context, code, redirectURI, codeVerifier string) (*twitter.Tokens, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeTwitterClient) GetUserInfo(ctx context.Context, accessToken string) (*twitter.UserInfo, error) {
	return nil, errors.New("not implemented")
}

func (c *fakeTwitterClient) RefreshToken(ctx context.Context, refreshToken string) (*twitter.Tokens, error) {
	c.refreshCalls = append(c.refreshCalls, refreshToken)
	if c.refreshErr != nil {
		return nil, c.refreshErr
	}
	return c.refreshTokens, nil
}

func (c *fakeTwitterClient) PostTweet(ctx context.Context, accessToken, text string) error {
	c.postedTokens = append(c.postedTokens, accessToken)
	if len(c.postErrs) == 0 {
		return nil
	}
	err := c.postErrs[0]
	c.postErrs = c.postErrs[1:]
	return err
}

func TestTwitterTokenNeedsRefresh(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"valid for another hour", now.Add(time.Hour), false},
		{"valid beyond the leeway", now.Add(twitterTokenLeeway + time.Minute), false},
		{"inside the leeway", now.Add(twitterTokenLeeway - time.Minute), true},
		{"already expired", now.Add(-time.Minute), true},
		{"expiry unknown", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, twitterTokenNeedsRefresh(tt.expiresAt, now))
		})
	}
}

func TestPostTweetWithRefresh_ValidTokenIsUsedAsIs(t *testing.T) {
	s := &fakeTwitterTokenStore{tokens: store.TwitterTokens{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}}
	c := &fakeTwitterClient{}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	require.NoError(t, err)
	assert.Equal(t, []string{"access"}, c.postedTokens)
	assert.Empty(t, c.refreshCalls)
	assert.Zero(t, s.writes)
}

func TestPostTweetWithRefresh_ExpiredTokenIsRefreshedAndStored(t *testing.T) {
	s := &fakeTwitterTokenStore{tokens: store.TwitterTokens{
		AccessToken:  "expired",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}}
	c := &fakeTwitterClient{refreshTokens: &twitter.Tokens{
		AccessToken:  "fresh",
		RefreshToken: "rotated",
		ExpiresIn:    7200,
	}}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	require.NoError(t, err)
	assert.Equal(t, []string{"refresh"}, c.refreshCalls)
	assert.Equal(t, []string{"fresh"}, c.postedTokens)
	assert.Equal(t, 1, s.writes)
	assert.Equal(t, "fresh", s.tokens.AccessToken)
	assert.Equal(t, "rotated", s.tokens.RefreshToken)
	assert.True(t, s.tokens.ExpiresAt.After(time.Now().Add(time.Hour)))
}

func TestPostTweetWithRefresh_RetriesOnceOn401(t *testing.T) {
	s := &fakeTwitterTokenStore{tokens: store.TwitterTokens{
		AccessToken:  "stale",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour), // looks valid, Twitter disagrees
	}}
	c := &fakeTwitterClient{
		postErrs:      []error{&twitter.APIError{StatusCode: http.StatusUnauthorized}},
		refreshTokens: &twitter.Tokens{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 7200},
	}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	require.NoError(t, err)
	assert.Equal(t, []string{"stale", "fresh"}, c.postedTokens)
	assert.Equal(t, 1, len(c.refreshCalls))
}

func TestPostTweetWithRefresh_DoesNotRetryTwice(t *testing.T) {
	s := &fakeTwitterTokenStore{tokens: store.TwitterTokens{
		AccessToken:  "stale",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}}
	unauthorized := &twitter.APIError{StatusCode: http.StatusUnauthorized}
	c := &fakeTwitterClient{
		postErrs:      []error{unauthorized, unauthorized},
		refreshTokens: &twitter.Tokens{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 7200},
	}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	require.Error(t, err)
	apiErr, ok := twitter.IsAPIError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Len(t, c.postedTokens, 2)
	assert.Len(t, c.refreshCalls, 1)
}

func TestPostTweetWithRefresh_SkipsRefreshWhenAnotherRequestAlreadyDidIt(t *testing.T) {
	s := &fakeTwitterTokenStore{tokens: store.TwitterTokens{
		AccessToken:  "stale",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}}
	c := &fakeTwitterClient{postErrs: []error{&twitter.APIError{StatusCode: http.StatusUnauthorized}}}

	// After the 401, a concurrent post has already refreshed the tokens.
	firstCall := true
	s.beforeRefresh = func(s *fakeTwitterTokenStore) {
		if firstCall {
			firstCall = false
			return
		}
		s.tokens = store.TwitterTokens{
			AccessToken:  "refreshed-elsewhere",
			RefreshToken: "rotated",
			ExpiresAt:    time.Now().Add(time.Hour),
		}
	}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	require.NoError(t, err)
	assert.Empty(t, c.refreshCalls)
	assert.Equal(t, []string{"stale", "refreshed-elsewhere"}, c.postedTokens)
}

func TestPostTweetWithRefresh_ReauthRequiredIsPropagated(t *testing.T) {
	s := &fakeTwitterTokenStore{tokens: store.TwitterTokens{
		AccessToken:  "expired",
		RefreshToken: "revoked",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}}
	c := &fakeTwitterClient{refreshErr: fmt.Errorf("%w: invalid_grant", twitter.ErrReauthRequired)}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	assert.ErrorIs(t, err, twitter.ErrReauthRequired)
	assert.Empty(t, c.postedTokens)
	assert.Zero(t, s.writes)
}

func TestPostTweetWithRefresh_NotConnected(t *testing.T) {
	s := &fakeTwitterTokenStore{err: store.ErrTwitterNotConnected}
	c := &fakeTwitterClient{}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	assert.ErrorIs(t, err, store.ErrTwitterNotConnected)
	assert.Empty(t, c.postedTokens)
}

func TestPostTweetWithRefresh_OtherAPIErrorIsNotRetried(t *testing.T) {
	s := &fakeTwitterTokenStore{tokens: store.TwitterTokens{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}}
	c := &fakeTwitterClient{postErrs: []error{&twitter.APIError{StatusCode: http.StatusTooManyRequests}}}

	err := postTweetWithRefresh(context.Background(), s, c, uuid.New(), "now playing")

	require.Error(t, err)
	apiErr, ok := twitter.IsAPIError(err)
	require.True(t, ok)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.Len(t, c.postedTokens, 1)
	assert.Empty(t, c.refreshCalls)
}

// *store.Store must satisfy the interface postTweetWithRefresh depends on.
var _ twitterTokenStore = (*store.Store)(nil)
