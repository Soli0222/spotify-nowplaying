package modules

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
)

type Artist struct {
	Name string `json:"name"`
}

type ExternalUrls struct {
	Spotify string `json:"spotify"`
}

type Item struct {
	Artists []Artist `json:"artists"`
	Name    string   `json:"name"`
	Show    struct {
		Name string `json:"name"`
	} `json:"show"`
	ExternalUrls ExternalUrls `json:"external_urls"`
}

type Data struct {
	CurrentlyPlayingType string `json:"currently_playing_type"`
	Item                 Item   `json:"item"`
}

type TrackData struct {
	TrackName  string
	TrackURL   string
	ArtistName string
	TrackEnc   string
}

type ShareURL struct {
	PSRTrack string
}

func GetReturnURL(resp *resty.Response, platform string) string {
	var shareURLData string
	var trackData TrackData
	bodyBytes := resp.Body()
	var data Data
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		fmt.Println("Error decoding JSON:", err)
	}

	if data.CurrentlyPlayingType == "track" {
		trackArtists := make([]string, 0)
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
	} else if data.CurrentlyPlayingType == "episode" {
		trackData = TrackData{
			TrackName:  data.Item.Name,
			TrackURL:   data.Item.ExternalUrls.Spotify,
			ArtistName: data.Item.Show.Name,
			TrackEnc:   url.QueryEscape(fmt.Sprintf("%s / %s\n#NowPlaying", data.Item.Name, data.Item.Show.Name)),
		}
	}

	if platform == "Misskey" {
		shareURLData = fmt.Sprintf("https://mi.soli0222.com/share?url=%s&text=%s", trackData.TrackURL, trackData.TrackEnc)
	} else if platform == "Twitter" {
		shareURLData = fmt.Sprintf("https://x.com/intent/tweet?url=%s&text=%s", trackData.TrackURL, trackData.TrackEnc)
	}

	return shareURLData
}
