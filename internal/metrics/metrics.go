package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPリクエスト関連
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// アプリケーション固有メトリクス
	SpotifyAPIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "spotify_api_requests_total",
			Help: "Total number of Spotify API requests",
		},
		[]string{"endpoint", "status"},
	)

	SpotifyAPIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "spotify_api_request_duration_seconds",
			Help:    "Spotify API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	ShareRedirectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "share_redirects_total",
			Help: "Total number of share redirects by platform and content type",
		},
		[]string{"platform", "content_type"},
	)

	OAuthCallbacksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oauth_callbacks_total",
			Help: "Total number of OAuth callbacks",
		},
		[]string{"platform", "status"},
	)
)
