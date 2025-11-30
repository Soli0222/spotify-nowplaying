package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/handler"
	"github.com/Soli0222/spotify-nowplaying/internal/metrics"
	"github.com/Soli0222/spotify-nowplaying/internal/spotify"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	if port == "" {
		port = "8080"
	}
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9090"
	}

	// メトリクスサーバーを別ポートで起動
	metricsServer := &http.Server{
		Addr:    ":" + metricsPort,
		Handler: promhttp.Handler(),
	}
	go func() {
		log.Printf("Metrics server starting on :%s", metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(metrics.PrometheusMiddleware())

	// SpotifyクライアントとHandlerを初期化
	spotifyClient := spotify.NewHTTPClient()
	h := handler.NewHandler(spotifyClient)

	// /は/noteにリダイレクト
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/note")
	})

	// statusハンドラー
	e.GET("/status", handler.StatusHandler)

	// /noteグループ
	noteGroup := e.Group("/note")
	noteGroup.GET("", handler.NoteLoginHandler)
	noteGroup.GET("/callback", h.NoteCallbackHandler)
	noteGroup.GET("/home", h.NoteHomeHandler)

	// /tweetグループ
	tweetGroup := e.Group("/tweet")
	tweetGroup.GET("", handler.TweetLoginHandler)
	tweetGroup.GET("/callback", h.TweetCallbackHandler)
	tweetGroup.GET("/home", h.TweetHomeHandler)

	// サーバーをゴルーチンで起動
	go func() {
		log.Printf("Main server starting on :%s", port)
		if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Main server failed: %v", err)
		}
	}()

	// シグナル待機
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	// グレースフルシャットダウン（タイムアウト10秒）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// メインサーバーのシャットダウン
	if err := e.Shutdown(ctx); err != nil {
		log.Printf("Main server shutdown error: %v", err)
	}

	// メトリクスサーバーのシャットダウン
	if err := metricsServer.Shutdown(ctx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}

	log.Println("Servers gracefully stopped")
}
