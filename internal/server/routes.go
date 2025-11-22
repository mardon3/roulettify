package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"roulettify/internal/auth"
	"roulettify/internal/game"
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

	// Guest auth route
	r.POST("/auth/guest", s.HandleGuestAuth)

	// WebSocket route
	r.GET("/ws", s.HandleWebSocket)

	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) HealthCheckHandler(c *gin.Context) {
	metrics := s.roomManager.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"metrics":   metrics,
	})
}

// HandleSpotifyAuth initiates the Spotify OAuth flow
func (s *Server) HandleSpotifyAuth(c *gin.Context) {
	state := uuid.New().String()
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	authURL := s.spotifyAuth.GetAuthURL(state)

	log.Printf("Redirecting to Spotify auth: %s", authURL)
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleSpotifyCallback handles the OAuth callback from Spotify
func (s *Server) HandleSpotifyCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	log.Printf("OAuth callback received - code: %s..., state: %s", code[:min(20, len(code))], state)

	storedState, err := c.Cookie("oauth_state")
	if err != nil {
		log.Printf("Cookie error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No state cookie found"})
		return
	}

	if storedState != state {
		log.Printf("State mismatch! Stored: %s, Received: %s", storedState, state)
		c.JSON(http.StatusBadRequest, gin.H{"error": "State mismatch"})
		return
	}

	log.Printf("State validation passed, exchanging code for token...")

	token, err := s.spotifyAuth.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code"})
		return
	}

	log.Printf("Token received: %s...", token.AccessToken[:20])

	spotifyClient := s.spotifyAuth.NewClient(c.Request.Context(), token)

	player, err := auth.FetchPlayerInfo(c.Request.Context(), spotifyClient)
	if err != nil {
		log.Printf("Failed to fetch player info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player info"})
		return
	}

	log.Printf("Player info fetched: %s (ID: %s)", player.Name, player.ID)

	topTracks, err := auth.FetchPlayerTopTracks(c.Request.Context(), spotifyClient)
	if err != nil {
		log.Printf("Failed to fetch top tracks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top tracks"})
		return
	}

	log.Printf("Fetched %d top tracks", len(topTracks))

	player.AccessToken = token.AccessToken
	player.TopTracks = topTracks

	playerJSON, _ := json.Marshal(map[string]interface{}{
		"id":           player.ID,
		"name":         player.Name,
		"spotify_id":   player.SpotifyID,
		"access_token": token.AccessToken,
		"is_guest":     false,
	})

	c.SetCookie("oauth_state", "", -1, "/", "", false, true)
	c.SetCookie("player_session", string(playerJSON), 3600, "/", "", false, false)

	log.Printf("Redirecting to frontend with auth=success")

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/?auth=success")
}

// HandleGuestAuth creates a guest account
func (s *Server) HandleGuestAuth(c *gin.Context) {
	var req struct {
		GuestIndex int `json:"guest_index"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Generate mock player
	player := auth.GenerateMockPlayer(req.GuestIndex)

	log.Printf("Guest player created: %s (ID: %s)", player.Name, player.ID)

	playerJSON, _ := json.Marshal(map[string]interface{}{
		"id":           player.ID,
		"name":         player.Name,
		"spotify_id":   player.SpotifyID,
		"access_token": player.AccessToken,
		"is_guest":     true,
	})

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"player_data":  string(playerJSON),
	})
}

// HandleWebSocket handles WebSocket connections for the game
func (s *Server) HandleWebSocket(c *gin.Context) {
	w := c.Writer
	r := c.Request

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := context.Background()
	var currentRoom *game.GameRoom
	var currentPlayer *game.Player

	// Message handling loop
	for {
		var msg game.Message
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		switch msg.Type {
		case game.MsgTypeJoinRoom:
			currentRoom, currentPlayer = s.handleJoinRoom(ctx, conn, msg.Payload)
			
		case game.MsgTypeStartGame:
			s.handleStartGame(currentRoom, msg.Payload)
			
		case game.MsgTypeSubmitGuess:
			s.handleSubmitGuess(currentRoom, currentPlayer, msg.Payload)
		}
	}

	// Clean up on disconnect
	if currentRoom != nil && currentPlayer != nil {
		currentRoom.Leave <- currentPlayer.ID
	}
}

func (s *Server) handleJoinRoom(ctx context.Context, conn *websocket.Conn, payload interface{}) (*game.GameRoom, *game.Player) {
	data, _ := json.Marshal(payload)
	var joinPayload game.JoinRoomPayload
	json.Unmarshal(data, &joinPayload)

	// Get or create room
	room, exists := s.roomManager.GetRoom(joinPayload.RoomID)
	if !exists {
		var err error
		room, err = s.roomManager.CreateRoom(joinPayload.RoomID)
		if err != nil {
			log.Printf("Failed to create room: %v", err)
			return nil, nil
		}
	}

	// Create player
	var authPlayer *auth.Player
	
	if joinPayload.IsGuest {
		// Extract guest index from player ID
		guestIndex := 0
		fmt.Sscanf(joinPayload.PlayerID, "guest_%d", &guestIndex)
		authPlayer = auth.GenerateMockPlayer(guestIndex)
	} else {
		// Fetch real player data
		spotifyClient := s.spotifyAuth.NewClient(ctx, &oauth2.Token{
			AccessToken: joinPayload.AccessToken,
		})
		
		var err error
		authPlayer, err = auth.FetchPlayerInfo(ctx, spotifyClient)
		if err != nil {
			log.Printf("Failed to fetch player info: %v", err)
			return nil, nil
		}
		
		tracks, err := auth.FetchPlayerTopTracks(ctx, spotifyClient)
		if err != nil {
			log.Printf("Failed to fetch top tracks: %v", err)
			return nil, nil
		}
		authPlayer.TopTracks = tracks
		authPlayer.AccessToken = joinPayload.AccessToken
	}

	player := &game.Player{
		Player:     authPlayer,
		Connection: conn,
		JoinedAt:   time.Now(),
	}

	// Join room
	room.Join <- player

	return room, player
}

func (s *Server) handleStartGame(room *game.GameRoom, payload interface{}) {
	if room == nil {
		return
	}

	data, _ := json.Marshal(payload)
	var startPayload game.StartGamePayload
	json.Unmarshal(data, &startPayload)

	totalRounds := startPayload.TotalRounds
	if totalRounds <= 0 {
		totalRounds = 10
	}

	room.StartGame <- totalRounds
}

func (s *Server) handleSubmitGuess(room *game.GameRoom, player *game.Player, payload interface{}) {
	if room == nil || player == nil {
		return
	}

	data, _ := json.Marshal(payload)
	var guessPayload game.SubmitGuessPayload
	json.Unmarshal(data, &guessPayload)

	room.Guess <- game.Guess{
		PlayerID:        player.ID,
		GuessedPlayerID: guessPayload.GuessedPlayerID,
		Timestamp:       time.Now(),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}