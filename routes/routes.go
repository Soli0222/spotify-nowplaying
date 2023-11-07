package routes

import (
	"net/http"
	"spotify-nowplaying/handlers"
)

func SetupRoutes() *http.ServeMux {
	r := http.NewServeMux()

	// ルートハンドラーをマップします
	r.HandleFunc("/note", handlers.NoteLoginHandler)
	r.HandleFunc("/note/callback", handlers.NoteCallbackHandler)
	r.HandleFunc("/note/home", handlers.NoteHomeHandler)
	r.HandleFunc("/tweet", handlers.TweetLoginHandler)
	r.HandleFunc("/tweet/callback", handlers.TweetCallbackHandler)
	r.HandleFunc("/tweet/home", handlers.TweetHomeHandler)

	// ルートパスのリダイレクト
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/note", http.StatusFound)
	})

	return r
}
