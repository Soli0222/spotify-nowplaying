package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Soli0222/spotify-nowplaying/modules"

	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
)

func TweetHomeHandler(c echo.Context) error {
	cookie, err := c.Cookie("access_token")
	if err != nil {
		return err
	}

	access_token := cookie.Value
	spotifyEndpoint := "https://api.spotify.com/v1/me/player?market=JP"

	resp, err := resty.New().R().
		SetHeader("Authorization", "Bearer "+access_token).
		SetHeader("Accept-Language", "ja").
		Get(spotifyEndpoint)

	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	redirectURL := (modules.GetReturnURL(resp, "Twitter"))

	statusCode := resp.StatusCode()
	if statusCode != 200 {
		statusText := http.StatusText(statusCode)
		return c.String(http.StatusOK, fmt.Sprintf("%d", statusCode)+": "+statusText)
	}

	return c.Redirect(http.StatusFound, redirectURL)

}

func TweetLoginHandler(c echo.Context) error {
	authURL := "https://accounts.spotify.com/authorize"
	scope := "user-read-currently-playing user-read-playback-state"
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI_TWEET")

	redirectURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s", authURL, clientID, redirectURI, scope)

	return c.Redirect(http.StatusFound, redirectURL)
}

func TweetCallbackHandler(c echo.Context) error {
	code := c.QueryParam("code")
	fmt.Println("callback")

	if code != "" {
		modules.CallbackHandler(c, code, "Twitter")
		return c.Redirect(http.StatusFound, "/tweet/home")
	} else {
		return c.String(http.StatusBadRequest, "Code parameter is missing.")
	}
}
