package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostTargetConstants(t *testing.T) {
	assert.Equal(t, PostTarget("misskey"), PostTargetMisskey)
	assert.Equal(t, PostTarget("twitter"), PostTargetTwitter)
	assert.Equal(t, PostTarget("both"), PostTargetBoth)
}

func TestPostResponse_JSONMarshal(t *testing.T) {
	resp := PostResponse{
		Success: true,
		Message: "test message",
		Results: map[string]string{
			"misskey": "success",
			"twitter": "error",
		},
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var unmarshaled PostResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, resp.Success, unmarshaled.Success)
	assert.Equal(t, resp.Message, unmarshaled.Message)
	assert.Equal(t, resp.Results, unmarshaled.Results)
}

func TestMisskeyNoteRequest_JSONMarshal(t *testing.T) {
	req := MisskeyNoteRequest{
		I:          "token123",
		Text:       "Hello World",
		Visibility: "public",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled map[string]string
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, "token123", unmarshaled["i"])
	assert.Equal(t, "Hello World", unmarshaled["text"])
	assert.Equal(t, "public", unmarshaled["visibility"])
}

func TestTwitterTweetRequest_JSONMarshal(t *testing.T) {
	req := TwitterTweetRequest{
		Text: "Hello Twitter",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled map[string]string
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, "Hello Twitter", unmarshaled["text"])
}

func TestUserInfoResponse_JSONMarshal(t *testing.T) {
	resp := UserInfoResponse{
		ID:                    "user-123",
		SpotifyUserID:         "spotify-456",
		SpotifyDisplayName:    "Test User",
		SpotifyImageURL:       "https://example.com/image.jpg",
		MisskeyConnected:      true,
		MisskeyInstanceURL:    "https://misskey.tld",
		MisskeyUserID:         "misskey-789",
		MisskeyUsername:       "testuser",
		MisskeyAvatarURL:      "https://misskey.tld/avatar.jpg",
		MisskeyHost:           "misskey.tld",
		TwitterConnected:      true,
		TwitterUserID:         "twitter-101",
		TwitterUsername:       "twitteruser",
		TwitterAvatarURL:      "https://twitter.com/avatar.jpg",
		APIURLToken:           "api-token",
		APIHeaderTokenEnabled: true,
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	var unmarshaled UserInfoResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, resp.ID, unmarshaled.ID)
	assert.Equal(t, resp.SpotifyUserID, unmarshaled.SpotifyUserID)
	assert.Equal(t, resp.MisskeyConnected, unmarshaled.MisskeyConnected)
	assert.Equal(t, resp.TwitterConnected, unmarshaled.TwitterConnected)
	assert.Equal(t, resp.APIHeaderTokenEnabled, unmarshaled.APIHeaderTokenEnabled)
}

func TestUserInfoResponse_JSONOmitEmpty(t *testing.T) {
	resp := UserInfoResponse{
		ID:               "user-123",
		SpotifyUserID:    "spotify-456",
		MisskeyConnected: false,
		TwitterConnected: false,
		APIURLToken:      "token",
	}

	data, err := json.Marshal(resp)
	assert.NoError(t, err)

	jsonStr := string(data)

	// omitempty フィールドは空の場合含まれない
	assert.NotContains(t, jsonStr, "spotify_display_name")
	assert.NotContains(t, jsonStr, "misskey_instance_url")
	assert.NotContains(t, jsonStr, "twitter_user_id")

	// 必須フィールドは常に含まれる
	assert.Contains(t, jsonStr, "id")
	assert.Contains(t, jsonStr, "spotify_user_id")
	assert.Contains(t, jsonStr, "misskey_connected")
	assert.Contains(t, jsonStr, "twitter_connected")
}
