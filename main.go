package main

import (
	"log"
	"net/http"
	"spotify-nowplaying/routes"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	r := routes.SetupRoutes()
	http.Handle("/", r)
	http.ListenAndServe(":3000", nil)
}
