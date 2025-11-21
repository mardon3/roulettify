package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"roulettify/internal/auth"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port        int
	spotifyAuth *auth.SpotifyAuthenticator
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	
	// Initialize Spotify authenticator
	spotifyAuth := auth.NewSpotifyAuthenticator(
		os.Getenv("SPOTIFY_CLIENT_ID"),
		os.Getenv("SPOTIFY_CLIENT_SECRET"),
		os.Getenv("SPOTIFY_REDIRECT_URI"),
	)

	NewServer := &Server{
		port:        port,
		spotifyAuth: spotifyAuth,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}