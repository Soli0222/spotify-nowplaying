package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func StatusHandler(c echo.Context) error {
	start := time.Now()

	statusCode := http.StatusOK

	response := map[string]interface{}{
		"status_code":   statusCode,
		"response_time": time.Since(start).Milliseconds(),
	}

	return c.JSON(statusCode, response)
}
