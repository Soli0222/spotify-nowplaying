package metrics

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// PrometheusMiddleware はHTTPリクエストのメトリクスを収集するミドルウェア
func PrometheusMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start).Seconds()
			status := c.Response().Status
			method := c.Request().Method
			path := c.Path()

			// パスが空の場合はリクエストURIを使用
			if path == "" {
				path = c.Request().URL.Path
			}

			HttpRequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
			HttpRequestDuration.WithLabelValues(method, path).Observe(duration)

			return err
		}
	}
}
