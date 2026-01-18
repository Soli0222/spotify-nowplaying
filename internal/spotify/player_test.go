package spotify

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetShareInfo_Track_Misskey(t *testing.T) {
	t.Setenv("SERVER_URI", "misskey.example.com")

	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "track",
		Item: Item{
			Name:    "Test Song",
			Artists: []Artist{{Name: "Test Artist"}},
			ExternalUrls: ExternalUrls{
				Spotify: "https://open.spotify.com/track/123",
			},
		},
	}

	shareURL, contentType := GetShareInfo(playerResp, "Misskey")

	assert.Equal(t, "track", contentType)
	assert.Contains(t, shareURL, "https://misskey.example.com/share")
	assert.Contains(t, shareURL, "url=https://open.spotify.com/track/123")
	assert.Contains(t, shareURL, url.QueryEscape("Test Song / Test Artist"))
	assert.Contains(t, shareURL, url.QueryEscape("#NowPlaying #PsrPlaying"))
}

func TestGetShareInfo_Track_Twitter(t *testing.T) {
	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "track",
		Item: Item{
			Name:    "Test Song",
			Artists: []Artist{{Name: "Artist1"}, {Name: "Artist2"}},
			ExternalUrls: ExternalUrls{
				Spotify: "https://open.spotify.com/track/456",
			},
		},
	}

	shareURL, contentType := GetShareInfo(playerResp, "Twitter")

	assert.Equal(t, "track", contentType)
	assert.Contains(t, shareURL, "https://x.com/intent/tweet")
	assert.Contains(t, shareURL, "url=https://open.spotify.com/track/456")
	assert.Contains(t, shareURL, url.QueryEscape("Test Song / Artist1, Artist2"))
}

func TestGetShareInfo_Episode(t *testing.T) {
	t.Setenv("SERVER_URI", "https://misskey.example.com")

	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "episode",
		Item: Item{
			Name: "Test Episode",
			Show: struct {
				Name string `json:"name"`
			}{Name: "Test Podcast"},
			ExternalUrls: ExternalUrls{
				Spotify: "https://open.spotify.com/episode/789",
			},
		},
	}

	shareURL, contentType := GetShareInfo(playerResp, "Misskey")

	assert.Equal(t, "episode", contentType)
	assert.Contains(t, shareURL, "https://misskey.example.com/share")
	assert.Contains(t, shareURL, url.QueryEscape("Test Episode / Test Podcast"))
	assert.Contains(t, shareURL, url.QueryEscape("#NowPlaying"))
}

func TestGetShareInfo_Unknown(t *testing.T) {
	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "ad",
		Item:                 Item{},
	}

	_, contentType := GetShareInfo(playerResp, "Twitter")

	assert.Equal(t, "unknown", contentType)
}

func TestGetShareInfo_NilResponse(t *testing.T) {
	shareURL, contentType := GetShareInfo(nil, "Twitter")

	assert.Equal(t, "unknown", contentType)
	assert.Empty(t, shareURL)
}

func TestGetShareInfo_ServerURIWithoutScheme(t *testing.T) {
	t.Setenv("SERVER_URI", "misskey.tld")

	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "track",
		Item: Item{
			Name:    "Song",
			Artists: []Artist{{Name: "Artist"}},
			ExternalUrls: ExternalUrls{
				Spotify: "https://open.spotify.com/track/abc",
			},
		},
	}

	shareURL, _ := GetShareInfo(playerResp, "Misskey")

	assert.Contains(t, shareURL, "https://misskey.tld/share")
}

func TestGetShareInfo_ServerURIWithHTTP(t *testing.T) {
	t.Setenv("SERVER_URI", "http://localhost:3000")

	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "track",
		Item: Item{
			Name:    "Song",
			Artists: []Artist{{Name: "Artist"}},
			ExternalUrls: ExternalUrls{
				Spotify: "https://open.spotify.com/track/abc",
			},
		},
	}

	shareURL, _ := GetShareInfo(playerResp, "Misskey")

	assert.Contains(t, shareURL, "http://localhost:3000/share")
}

func TestParsePlayerResponse_Track(t *testing.T) {
	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "track",
		Item: Item{
			Name:    "Test Song",
			Artists: []Artist{{Name: "Artist1"}, {Name: "Artist2"}},
			ExternalUrls: ExternalUrls{
				Spotify: "https://open.spotify.com/track/123",
			},
		},
	}

	trackData, contentType := ParsePlayerResponse(playerResp)

	assert.Equal(t, "track", contentType)
	assert.Equal(t, "Test Song", trackData.TrackName)
	assert.Equal(t, "https://open.spotify.com/track/123", trackData.TrackURL)
	assert.Equal(t, "Artist1, Artist2", trackData.ArtistName)
}

func TestParsePlayerResponse_Episode(t *testing.T) {
	playerResp := &PlayerResponse{
		CurrentlyPlayingType: "episode",
		Item: Item{
			Name: "Test Episode",
			Show: struct {
				Name string `json:"name"`
			}{Name: "Test Podcast"},
			ExternalUrls: ExternalUrls{
				Spotify: "https://open.spotify.com/episode/789",
			},
		},
	}

	trackData, contentType := ParsePlayerResponse(playerResp)

	assert.Equal(t, "episode", contentType)
	assert.Equal(t, "Test Episode", trackData.TrackName)
	assert.Equal(t, "Test Podcast", trackData.ArtistName)
}

func TestBuildShareURL_Misskey(t *testing.T) {
	t.Setenv("SERVER_URI", "misskey.tld")

	trackData := TrackData{
		TrackName:  "Song",
		TrackURL:   "https://open.spotify.com/track/abc",
		ArtistName: "Artist",
		TrackEnc:   url.QueryEscape("Song / Artist\n#NowPlaying #PsrPlaying"),
	}

	shareURL := BuildShareURL(trackData, "Misskey")

	assert.Contains(t, shareURL, "https://misskey.tld/share")
	assert.Contains(t, shareURL, "url=https://open.spotify.com/track/abc")
}

func TestBuildShareURL_Twitter(t *testing.T) {
	trackData := TrackData{
		TrackName:  "Song",
		TrackURL:   "https://open.spotify.com/track/abc",
		ArtistName: "Artist",
		TrackEnc:   url.QueryEscape("Song / Artist\n#NowPlaying"),
	}

	shareURL := BuildShareURL(trackData, "Twitter")

	assert.Contains(t, shareURL, "https://x.com/intent/tweet")
	assert.Contains(t, shareURL, "url=https://open.spotify.com/track/abc")
}

func TestBuildShareURL_UnknownPlatform(t *testing.T) {
	trackData := TrackData{
		TrackURL: "https://open.spotify.com/track/abc",
	}

	shareURL := BuildShareURL(trackData, "Unknown")

	assert.Empty(t, shareURL)
}
