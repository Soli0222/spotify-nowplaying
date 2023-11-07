package handlers

import (
	"fmt"
	"net/http"
	"os"
)

func TweetHomeHandler(w http.ResponseWriter, r *http.Request) {
	// /tweet/home へのリクエストを処理
	w.Write([]byte("tweet Home Page"))
}

func TweetLoginHandler(w http.ResponseWriter, r *http.Request) {
	authURL := "https://accounts.spotify.com/authorize"
	scope := "user-read-currently-playing user-read-playback-state"
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI_TWEET")

	redirectURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s", authURL, clientID, redirectURI, scope)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func TweetCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI_TWEET")
	tokenURL := "https://accounts.spotify.com/api/token"

	request, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	request.SetBasicAuth(clientID, clientSecret)

	params := request.URL.Query()
	params.Add("grant_type", "authorization_code")
	params.Add("code", code)
	params.Add("redirect_uri", redirectURI)
	request.URL.RawQuery = params.Encode()

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		http.Error(w, "Failed to make request", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	http.Redirect(w, r, "/tweet/home", http.StatusFound)
}
