package handler

import "github.com/labstack/echo/v4"

// TweetHomeHandler はTwitter向けのホームハンドラー
func (h *Handler) TweetHomeHandler(c echo.Context) error {
	return h.homeHandler(c, PlatformTwitter, "twitter")
}

// TweetLoginHandler はTwitter向けのログインハンドラー
func TweetLoginHandler(c echo.Context) error {
	return loginHandler(c, "/tweet/callback")
}

// TweetCallbackHandler はTwitter向けのコールバックハンドラー
func (h *Handler) TweetCallbackHandler(c echo.Context) error {
	return h.callbackHandler(c, PlatformTwitter, "twitter", "/tweet/home")
}
