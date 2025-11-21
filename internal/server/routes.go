package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"roulettify/internal/auth"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Basic routes
	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.HealthCheckHandler)

	// Spotify OAuth routes
	r.GET("/auth/spotify", s.HandleSpotifyAuth)
	r.GET("/auth/callback", s.HandleSpotifyCallback)

	// WebSocket route
	r.GET("/websocket", s.websocketHandler)

	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

// HandleSpotifyAuth initiates the Spotify OAuth flow
func (s *Server) HandleSpotifyAuth(c *gin.Context) {
	// Generate a random state for CSRF protection
	state := uuid.New().String()

	// Store state in cookie
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	// Get auth URL from Spotify
	authURL := s.spotifyAuth.GetAuthURL(state)

	log.Printf("Redirecting to Spotify auth: %s", authURL)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleSpotifyCallback handles the OAuth callback from Spotify
func (s *Server) HandleSpotifyCallback(c *gin.Context) {
	// Get code and state from query parameters
	code := c.Query("code")
	state := c.Query("state")

	log.Printf("OAuth callback received - code: %s..., state: %s", code[:20], state)

	// Verify state matches
	storedState, err := c.Cookie("oauth_state")
	if err != nil {
		log.Printf("Cookie error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No state cookie found",
		})
		return
	}

	if storedState != state {
		log.Printf("State mismatch! Stored: %s, Received: %s", storedState, state)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "State mismatch",
		})
		return
	}

	log.Printf("State validation passed, exchanging code for token...")

	// Exchange code for token
	token, err := s.spotifyAuth.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to exchange code",
		})
		return
	}

	log.Printf("Token received: %s...", token.AccessToken[:20])

	// Create Spotify client
	spotifyClient := s.spotifyAuth.NewClient(c.Request.Context(), token)

	// Fetch player info
	player, err := auth.FetchPlayerInfo(c.Request.Context(), spotifyClient)
	if err != nil {
		log.Printf("Failed to fetch player info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch player info",
		})
		return
	}

	log.Printf("Player info fetched: %s (ID: %s)", player.Name, player.ID)

	// Fetch top tracks
	topTracks, err := auth.FetchPlayerTopTracks(c.Request.Context(), spotifyClient)
	if err != nil {
		log.Printf("Failed to fetch top tracks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch top tracks",
		})
		return
	}

	log.Printf("Fetched %d top tracks", len(topTracks))

	player.AccessToken = token.AccessToken
	player.TopTracks = topTracks

	// Store in session cookie
	playerJSON, _ := json.Marshal(map[string]interface{}{
		"id":           player.ID,
		"name":         player.Name,
		"spotify_id":   player.SpotifyID,
		"access_token": token.AccessToken,
	})

	// Clear old state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	// Set new cookie with httpOnly = false so JavaScript can read it
	c.SetCookie("player_session", string(playerJSON), 3600, "/", "", false, false)

	log.Printf("Redirecting to frontend with auth=success")

	// Get frontend URL from environment or use default
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	// Redirect to frontend with success indicator
	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/?auth=success")
}

func (s *Server) websocketHandler(c *gin.Context) {
	w := c.Writer
	r := c.Request

	socket, err := websocket.Accept(w, r, nil)

	if err != nil {
		log.Printf("could not open websocket: %v", err)
		c.String(http.StatusInternalServerError, "could not open websocket")
		return
	}

	defer socket.Close(websocket.StatusGoingAway, "server closing websocket")

	ctx := r.Context()
	socketCtx := socket.CloseRead(ctx)

	for {
		payload := fmt.Sprintf("server timestamp: %d", time.Now().UnixNano())
		err := socket.Write(socketCtx, websocket.MessageText, []byte(payload))
		if err != nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
}