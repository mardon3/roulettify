package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"roulettify/internal/auth"
	"roulettify/internal/game"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port        int
	spotifyAuth *auth.SpotifyAuthenticator
	roomManager *game.RoomManager
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	
	// Initialize Spotify authenticator
	spotifyAuth := auth.NewSpotifyAuthenticator(
		os.Getenv("SPOTIFY_CLIENT_ID"),
		os.Getenv("SPOTIFY_CLIENT_SECRET"),
		os.Getenv("SPOTIFY_REDIRECT_URI"),
	)

	// Initialize game room manager
	roomManager := game.NewRoomManager()

	NewServer := &Server{
		port:        port,
		spotifyAuth: spotifyAuth,
		roomManager: roomManager,
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