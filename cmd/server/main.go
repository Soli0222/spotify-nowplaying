package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Soli0222/spotify-nowplaying/internal/auth"
	"github.com/Soli0222/spotify-nowplaying/internal/handler"
	"github.com/Soli0222/spotify-nowplaying/internal/metrics"
	"github.com/Soli0222/spotify-nowplaying/internal/spotify"
	"github.com/Soli0222/spotify-nowplaying/internal/store"

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
		"BASE_URL",
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

	// カスタムロガーミドルウェア（ステータスコードに応じてログレベルを変更）
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogHost:     true,
		LogError:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			level := "INFO"
			if v.Status >= 500 {
				level = "ERROR"
			} else if v.Status >= 400 {
				level = "WARN"
			}

			log.Printf(`{"time":"%s","level":"%s","remote_ip":"%s","host":"%s","method":"%s","uri":"%s","status":%d,"latency":"%s"}`,
				v.StartTime.Format(time.RFC3339Nano),
				level,
				v.RemoteIP,
				v.Host,
				v.Method,
				v.URI,
				v.Status,
				v.Latency.String(),
			)
			return nil
		},
	}))
	e.Use(middleware.Recover())
	e.Use(metrics.PrometheusMiddleware())

	// SpotifyクライアントとHandlerを初期化
	spotifyClient := spotify.NewHTTPClient()
	h := handler.NewHandler(spotifyClient)

	// Construct DATABASE_URL from POSTGRES_* env vars if not explicitly set
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		dbname := os.Getenv("POSTGRES_DB")

		if user != "" && host != "" && port != "" && dbname != "" {
			if password != "" {
				databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
			} else {
				databaseURL = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable", user, host, port, dbname)
			}
		}
	}

	// Database接続（オプション - databaseURLが設定されている場合のみ）
	var db *store.Store
	var jwtConfig auth.JWTConfig
	if databaseURL != "" {
		var err error
		db, err = store.New(databaseURL)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			}
		}()
		log.Println("Connected to database")

		jwtConfig = auth.DefaultJWTConfig()

		// API handlers
		spotifyAuthHandler := handler.NewSpotifyAuthHandler(db, spotifyClient, jwtConfig)
		miAuthHandler := handler.NewMiAuthHandler(db, jwtConfig)
		twitterAuthHandler := handler.NewTwitterAuthHandler(db, jwtConfig)
		settingsHandler := handler.NewSettingsHandler(db, jwtConfig)
		apiPostHandler := handler.NewAPIPostHandler(db, spotifyClient)

		// API routes
		api := e.Group("/api")

		// Public auth routes
		api.GET("/auth/check", spotifyAuthHandler.CheckAuth)
		api.GET("/auth/spotify", spotifyAuthHandler.LoginSpotify)
		api.GET("/auth/spotify/callback", spotifyAuthHandler.CallbackSpotify)

		// Public API post route (authenticated by URL token + optional header token)
		api.GET("/post/:token", apiPostHandler.PostNowPlaying)

		// MiAuth callback (no JWT required, uses session)
		api.GET("/miauth/callback", miAuthHandler.CallbackMiAuth)

		// Twitter callback (no JWT required, uses session)
		api.GET("/twitter/callback", twitterAuthHandler.CallbackTwitterAuth)

		// Protected routes (require JWT)
		protected := api.Group("")
		protected.Use(auth.JWTMiddleware(jwtConfig))

		// User info
		protected.GET("/me", settingsHandler.GetUserInfo)
		protected.POST("/logout", settingsHandler.Logout)

		// App config (requires auth for eligibility check)
		protected.GET("/config", settingsHandler.GetAppConfig)

		// MiAuth
		protected.POST("/miauth/start", miAuthHandler.StartMiAuth)
		protected.DELETE("/miauth", miAuthHandler.DisconnectMisskey)

		// Twitter
		protected.GET("/twitter/start", twitterAuthHandler.StartTwitterAuth)
		protected.DELETE("/twitter", twitterAuthHandler.DisconnectTwitter)

		// Settings
		protected.POST("/settings/header-token", settingsHandler.GenerateHeaderToken)
		protected.DELETE("/settings/header-token", settingsHandler.DisableHeaderToken)
		protected.POST("/settings/api-url-token/regenerate", settingsHandler.RegenerateAPIURLToken)

		// Serve SPA static files
		e.Static("/assets", "frontend/dist/assets")
		e.File("/vite.svg", "frontend/dist/vite.svg")

		// SPA fallback for frontend routes
		e.GET("/login", serveSPA)
		e.GET("/dashboard", serveSPA)
	}

	// /は/noteにリダイレクト (既存の動作を維持、ただしDBがある場合はフロントエンドへ)
	e.GET("/", func(c echo.Context) error {
		if db != nil {
			return serveSPA(c)
		}
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

// serveSPA serves the SPA index.html for frontend routes
func serveSPA(c echo.Context) error {
	return c.File("frontend/dist/index.html")
}
