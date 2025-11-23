package server

import (
	"context"
	"encoding/json"
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
	r.GET("/rooms", s.ListRoomsHandler)

	// Spotify OAuth routes
	r.GET("/auth/spotify", s.HandleSpotifyAuth)
	r.GET("/auth/callback", s.HandleSpotifyCallback)

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

func (s *Server) ListRoomsHandler(c *gin.Context) {
	rooms := s.roomManager.ListRooms()
	c.JSON(http.StatusOK, gin.H{
		"rooms": rooms,
	})
}

// HandleSpotifyAuth initiates the Spotify OAuth flow
func (s *Server) HandleSpotifyAuth(c *gin.Context) {
	state := uuid.New().String()
	c.SetCookie("oauth_state", state, 600, "/", "", false, true)

	authURL := s.spotifyAuth.GetAuthURL(state)

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

// HandleSpotifyCallback handles the OAuth callback from Spotify
func (s *Server) HandleSpotifyCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	storedState, err := c.Cookie("oauth_state")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No state cookie found"})
		return
	}

	if storedState != state {
		c.JSON(http.StatusBadRequest, gin.H{"error": "State mismatch"})
		return
	}

	token, err := s.spotifyAuth.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		log.Printf("Token exchange failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange code"})
		return
	}

	spotifyClient := s.spotifyAuth.NewClient(c.Request.Context(), token)

	player, err := auth.FetchPlayerInfo(c.Request.Context(), spotifyClient)
	if err != nil {
		log.Printf("Failed to fetch player info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player info"})
		return
	}

	topTracks, err := auth.FetchPlayerTopTracks(c.Request.Context(), spotifyClient)
	if err != nil {
		log.Printf("Failed to fetch top tracks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch top tracks"})
		return
	}

	player.AccessToken = token.AccessToken
	player.TopTracks = topTracks

	playerJSON, _ := json.Marshal(map[string]interface{}{
		"id":           player.ID,
		"name":         player.Name,
		"spotify_id":   player.SpotifyID,
		"access_token": token.AccessToken,
	})

	c.SetCookie("oauth_state", "", -1, "/", "", false, true)
	c.SetCookie("player_session", string(playerJSON), 3600, "/", "", false, false)

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/?auth=success")
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

	// Get persistent room (no creation, only 3 rooms exist)
	room, err := s.roomManager.GetRoom(joinPayload.RoomID)
	if err != nil {
		log.Printf("Failed to get room: %v", err)
		// Send error to client
		errorMsg := game.Message{
			Type: game.MsgTypeError,
			Payload: map[string]interface{}{
				"message": err.Error(),
			},
		}
		if sendErr := wsjson.Write(ctx, conn, errorMsg); sendErr != nil {
			log.Printf("Failed to send error message: %v", sendErr)
		}
		return nil, nil
	}

	// Create player - fetch real player data from Spotify
	spotifyClient := s.spotifyAuth.NewClient(ctx, &oauth2.Token{
		AccessToken: joinPayload.AccessToken,
	})
	
	authPlayer, err := auth.FetchPlayerInfo(ctx, spotifyClient)
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

	player := &game.Player{
		Player:     authPlayer,
		Connection: conn,
		JoinedAt:   time.Now(),
	}

	// Join the persistent room (no shutdown check needed)
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