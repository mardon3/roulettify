package game

import (
	"time"

	"roulettify/internal/auth"

	"github.com/coder/websocket"
)

// Player wraps auth.Player for game use
type Player struct {
	*auth.Player
	Connection *websocket.Conn
	JoinedAt   time.Time
	IsReady    bool
	IsLeader   bool
}

// GameState represents the current state of the game
type GameState string

const (
	StateWaiting  GameState = "waiting"
	StatePlaying  GameState = "playing"
	StateRoundEnd GameState = "round_end"
	StateGameOver GameState = "game_over"
)

// MessageType defines WebSocket message types
type MessageType string

const (
	// Client to Server
	MsgTypeJoinRoom     MessageType = "join_room"
	MsgTypeLeaveRoom    MessageType = "leave_room"
	MsgTypeReady        MessageType = "ready"
	MsgTypeStartGame    MessageType = "start_game"
	MsgTypeSubmitGuess  MessageType = "submit_guess"

	// Server to Client
	MsgTypePlayerJoined   MessageType = "player_joined"
	MsgTypePlayerLeft     MessageType = "player_left"
	MsgTypePlayerReady    MessageType = "player_ready"
	MsgTypeGameStarted    MessageType = "game_started"
	MsgTypeRoundStarted   MessageType = "round_started"
	MsgTypeGuessReceived  MessageType = "guess_received"
	MsgTypeRoundComplete  MessageType = "round_complete"
	MsgTypeGameOver       MessageType = "game_over"
	MsgTypeGameReset      MessageType = "game_reset"
	MsgTypeError          MessageType = "error"
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// JoinRoomPayload for joining a room
type JoinRoomPayload struct {
	RoomID      string `json:"room_id"`
	PlayerID    string `json:"player_id"`
	PlayerName  string `json:"player_name"`
	AccessToken string `json:"access_token"`
}

// ReadyPayload for readying up
type ReadyPayload struct {
	PlayerID string `json:"player_id"`
	IsReady  bool   `json:"is_ready"`
}

// StartGamePayload for starting a game
type StartGamePayload struct {
	RoomID      string `json:"room_id"`
	TotalRounds int    `json:"total_rounds"`
}

// SubmitGuessPayload for submitting a guess
type SubmitGuessPayload struct {
	RoomID          string `json:"room_id"`
	PlayerID        string `json:"player_id"`
	GuessedPlayerID string `json:"guessed_player_id"`
}

// Guess represents a player's guess
type Guess struct {
	PlayerID        string    `json:"player_id"`
	GuessedPlayerID string    `json:"guessed_player_id"`
	Timestamp       time.Time `json:"timestamp"`
}

// RoundResult contains the results of a round
type RoundResult struct {
	Round           int                    `json:"round"`
	Track           auth.Track             `json:"track"`
	WinnerID        string                 `json:"winner_id"`
	WinnerRank      int                    `json:"winner_rank"`
	CorrectGuessers []string               `json:"correct_guessers"`
	PointsAwarded   map[string]int         `json:"points_awarded"`
	AllRankings     map[string]int         `json:"all_rankings"`
	UpdatedScores   map[string]int         `json:"updated_scores"`
	GuessDurations  map[string]float64     `json:"guess_durations"`
}

// PlayerInfo for client-side display
type PlayerInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Score    int    `json:"score"`
	IsReady  bool   `json:"is_ready"`
	IsLeader bool   `json:"is_leader"`
}