package modules

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

type (
	Tokens struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		ExpiresIn    int    `json:"expire_in"`
		RefreshToken string `json:"refresh_token"`
	}
)

func CallbackHandler(c echo.Context, code string, platform string) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	var redirectURI string
	if platform == "Misskey" {
		redirectURI = os.Getenv("SPOTIFY_REDIRECT_URI_NOTE")
	} else if platform == "Twitter" {
		redirectURI = os.Getenv("SPOTIFY_REDIRECT_URI_TWEET")
	}

	tokenURL := "https://accounts.spotify.com/api/token"

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Add("code", code)
	form.Add("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		fmt.Println(err)
		return
	}

	auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic "+auth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	byteArray, _ := io.ReadAll(resp.Body)
	data := new(Tokens) // bind the json
	if err := json.Unmarshal(byteArray, data); err != nil {
		fmt.Println(err)
		return
	}

	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    data.AccessToken,
		HttpOnly: true,
	}
	http.SetCookie(c.Response(), cookie)
}
