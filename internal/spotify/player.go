package spotify

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Artist はSpotifyのアーティスト情報
type Artist struct {
	Name string `json:"name"`
}

// ExternalUrls は外部URL情報
type ExternalUrls struct {
	Spotify string `json:"spotify"`
}

// Item はSpotifyの再生アイテム情報
type Item struct {
	Artists []Artist `json:"artists"`
	Name    string   `json:"name"`
	Show    struct {
		Name string `json:"name"`
	} `json:"show"`
	ExternalUrls ExternalUrls `json:"external_urls"`
}

// TrackData はシェア用のトラック情報
type TrackData struct {
	TrackName  string
	TrackURL   string
	ArtistName string
	TrackEnc   string
}

// ParsePlayerResponse はPlayerResponseからシェア情報を生成する
func ParsePlayerResponse(data *PlayerResponse) (TrackData, string) {
	var trackData TrackData
	var contentType string

	if data == nil {
		return trackData, "unknown"
	}

	switch data.CurrentlyPlayingType {
	case "track":
		contentType = "track"
		trackArtists := make([]string, 0, len(data.Item.Artists))
		for _, artist := range data.Item.Artists {
			trackArtists = append(trackArtists, artist.Name)
		}
		trackArtist := strings.Join(trackArtists, ", ")

		trackData = TrackData{
			TrackName:  data.Item.Name,
			TrackURL:   data.Item.ExternalUrls.Spotify,
			ArtistName: trackArtist,
			TrackEnc:   url.QueryEscape(fmt.Sprintf("%s / %s\n#NowPlaying #PsrPlaying", data.Item.Name, trackArtist)),
		}
	case "episode":
		contentType = "episode"
		trackData = TrackData{
			TrackName:  data.Item.Name,
			TrackURL:   data.Item.ExternalUrls.Spotify,
			ArtistName: data.Item.Show.Name,
			TrackEnc:   url.QueryEscape(fmt.Sprintf("%s / %s\n#NowPlaying", data.Item.Name, data.Item.Show.Name)),
		}
	default:
		contentType = "unknown"
	}

	return trackData, contentType
}

// BuildShareURL はプラットフォームに応じたシェアURLを生成する
func BuildShareURL(trackData TrackData, platform string) string {
	switch platform {
	case "Misskey":
		serverURI := os.Getenv("SERVER_URI")
		if !strings.HasPrefix(serverURI, "http://") && !strings.HasPrefix(serverURI, "https://") {
			serverURI = "https://" + serverURI
		}
		return fmt.Sprintf("%s/share?url=%s&text=%s", serverURI, trackData.TrackURL, trackData.TrackEnc)
	case "Twitter":
		return fmt.Sprintf("https://x.com/intent/tweet?url=%s&text=%s", trackData.TrackURL, trackData.TrackEnc)
	default:
		return ""
	}
}

// GetShareInfo はPlayerResponseからシェア情報を取得する
func GetShareInfo(data *PlayerResponse, platform string) (shareURL string, contentType string) {
	trackData, contentType := ParsePlayerResponse(data)
	if contentType == "unknown" {
		return "", contentType
	}
	shareURL = BuildShareURL(trackData, platform)
	return shareURL, contentType
}
