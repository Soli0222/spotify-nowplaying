package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusMiddleware(t *testing.T) {
	e := echo.New()
	e.Use(PrometheusMiddleware())

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestPrometheusMiddleware_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		path       string
	}{
		{"OK", http.StatusOK, "/ok"},
		{"NotFound", http.StatusNotFound, "/notfound"},
		{"InternalError", http.StatusInternalServerError, "/error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(PrometheusMiddleware())

			e.GET(tt.path, func(c echo.Context) error {
				return c.String(tt.statusCode, "")
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.statusCode, rec.Code)
		})
	}
}

func TestHttpRequestsTotal(t *testing.T) {
	require.NotNil(t, HttpRequestsTotal)

	assert.NotPanics(t, func() {
		HttpRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
	})
}

func TestHttpRequestDuration(t *testing.T) {
	require.NotNil(t, HttpRequestDuration)

	assert.NotPanics(t, func() {
		HttpRequestDuration.WithLabelValues("GET", "/test").Observe(0.5)
	})
}

func TestSpotifyAPIMetrics(t *testing.T) {
	require.NotNil(t, SpotifyAPIRequestsTotal)
	require.NotNil(t, SpotifyAPIRequestDuration)

	assert.NotPanics(t, func() {
		SpotifyAPIRequestsTotal.WithLabelValues("player", "200").Inc()
		SpotifyAPIRequestDuration.WithLabelValues("player").Observe(0.1)
	})
}

func TestShareRedirectsTotal(t *testing.T) {
	require.NotNil(t, ShareRedirectsTotal)

	assert.NotPanics(t, func() {
		ShareRedirectsTotal.WithLabelValues("misskey", "track").Inc()
		ShareRedirectsTotal.WithLabelValues("twitter", "episode").Inc()
	})
}

func TestOAuthCallbacksTotal(t *testing.T) {
	require.NotNil(t, OAuthCallbacksTotal)

	assert.NotPanics(t, func() {
		OAuthCallbacksTotal.WithLabelValues("misskey", "success").Inc()
		OAuthCallbacksTotal.WithLabelValues("twitter", "error").Inc()
	})
}
