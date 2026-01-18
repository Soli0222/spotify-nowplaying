package handler

import "github.com/labstack/echo/v4"

// NoteHomeHandler はMisskey向けのホームハンドラー
func (h *Handler) NoteHomeHandler(c echo.Context) error {
	return h.homeHandler(c, PlatformMisskey, "misskey")
}

// NoteLoginHandler はMisskey向けのログインハンドラー
func NoteLoginHandler(c echo.Context) error {
	return loginHandler(c, "/note/callback")
}

// NoteCallbackHandler はMisskey向けのコールバックハンドラー
func (h *Handler) NoteCallbackHandler(c echo.Context) error {
	return h.callbackHandler(c, PlatformMisskey, "misskey", "/note/home")
}
