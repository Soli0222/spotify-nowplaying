package main

import (
	"log"
	"net/http"
	"os"
	"spotify-nowplaying/handlers"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Not using .env file")
	}

	requiredVars := []string{
		"SERVER_URI",
		"SPOTIFY_CLIENT_ID",
		"SPOTIFY_CLIENT_SECRET",
		"SPOTIFY_REDIRECT_URI_NOTE",
		"SPOTIFY_REDIRECT_URI_TWEET",
	}

	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			log.Fatalf("Environment variable %s is not set", v)
		}
	}

	port := os.Getenv("PORT")

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// /は/noteにリダイレクト
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/note")
	})

	// statusハンドラー
	e.GET("/status", handlers.StatusHandler)

	// /noteグループ
	noteGroup := e.Group("/note")
	noteGroup.GET("", handlers.NoteLoginHandler)
	noteGroup.GET("/callback", handlers.NoteCallbackHandler)
	noteGroup.GET("/home", handlers.NoteHomeHandler)

	// /tweetグループ
	tweetGroup := e.Group("/tweet")
	tweetGroup.GET("", handlers.TweetLoginHandler)
	tweetGroup.GET("/callback", handlers.TweetCallbackHandler)
	tweetGroup.GET("/home", handlers.TweetHomeHandler)

	e.Logger.Fatal(e.Start(":" + port))
}
