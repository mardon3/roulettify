package game

import (
	"context"
	"log"
	"math/rand"
	"sort"
	"sync"
	"time"

	"roulettify/internal/auth"

	"github.com/coder/websocket/wsjson"
)

const MaxPlayersPerRoom = 6

type GameRoom struct {
	ID           string
	Players      map[string]*Player
	PlayerOrder  []string
	Scores       map[string]int
	CurrentRound int
	TotalRounds  int
	CurrentTrack *auth.Track
	Guesses      map[string]Guess
	State        GameState
	RoundTimer   *time.Timer

	// Channels
	Join      chan *Player
	Leave     chan string
	Guess     chan Guess
	StartGame chan int
	Broadcast chan Message

	mu sync.RWMutex
}

func NewGameRoom(id string) *GameRoom {
	return &GameRoom{
		ID:           id,
		Players:      make(map[string]*Player),
		PlayerOrder:  make([]string, 0),
		Scores:       make(map[string]int),
		Guesses:      make(map[string]Guess),
		State:        StateWaiting,
		Join:         make(chan *Player, 10),
		Leave:        make(chan string, 10),
		Guess:        make(chan Guess, 10),
		StartGame:    make(chan int, 1),
		Broadcast:    make(chan Message, 10),
	}
}

func (r *GameRoom) Run() {
	defer func() {
		if r.RoundTimer != nil {
			r.RoundTimer.Stop()
		}
		log.Printf("Room %s: Goroutine stopped", r.ID)
	}()

	for {
		select {
		case player := <-r.Join:
			r.handlePlayerJoin(player)

		case playerID := <-r.Leave:
			r.handlePlayerLeave(playerID)

		case totalRounds := <-r.StartGame:
			r.handleGameStart(totalRounds)

		case guess := <-r.Guess:
			r.handleGuess(guess)

		case msg := <-r.Broadcast:
			r.broadcastToAll(msg)
		}
	}
}

func (r *GameRoom) handlePlayerJoin(player *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check room capacity
	if len(r.Players) >= MaxPlayersPerRoom {
		log.Printf("Room %s is full (%d/%d players)", r.ID, len(r.Players), MaxPlayersPerRoom)
		r.Broadcast <- Message{
			Type: MsgTypeError,
			Payload: map[string]interface{}{
				"message": "Room is full (maximum 6 players)",
			},
		}
		return
	}

	// Add player
	r.Players[player.ID] = player
	r.PlayerOrder = append(r.PlayerOrder, player.ID)
	r.Scores[player.ID] = 0

	log.Printf("Player %s joined room %s", player.Name, r.ID)

	// Broadcast player joined
	r.Broadcast <- Message{
		Type: MsgTypePlayerJoined,
		Payload: map[string]interface{}{
			"player": PlayerInfo{
				ID:    player.ID,
				Name:  player.Name,
				Score: 0,
			},
			"player_count": len(r.Players),
			"players":      r.getPlayerInfoList(),
		},
	}
}

func (r *GameRoom) handlePlayerLeave(playerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.Players[playerID]
	if !exists {
		return
	}

	// Close WebSocket connection
	if player.Connection != nil {
		player.Connection.Close(1000, "Player left")
	}

	delete(r.Players, playerID)
	delete(r.Scores, playerID)
	delete(r.Guesses, playerID)

	// Remove from order
	for i, id := range r.PlayerOrder {
		if id == playerID {
			r.PlayerOrder = append(r.PlayerOrder[:i], r.PlayerOrder[i+1:]...)
			break
		}
	}

	log.Printf("Player %s left room %s", player.Name, r.ID)

	// Broadcast player left
	r.Broadcast <- Message{
		Type: MsgTypePlayerLeft,
		Payload: map[string]interface{}{
			"player_id":    playerID,
			"player_count": len(r.Players),
			"players":      r.getPlayerInfoList(),
		},
	}

	// If room becomes empty during a game, reset to waiting state
	if len(r.Players) == 0 && r.State != StateWaiting {
		r.State = StateWaiting
		r.CurrentRound = 0
		r.Scores = make(map[string]int)
		if r.RoundTimer != nil {
			r.RoundTimer.Stop()
		}
	}
}

func (r *GameRoom) handleGameStart(totalRounds int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StateWaiting {
		return
	}

	if len(r.Players) < 2 {
		r.Broadcast <- Message{
			Type: MsgTypeError,
			Payload: map[string]interface{}{
				"message": "Need at least 2 players to start",
			},
		}
		return
	}

	r.TotalRounds = totalRounds
	r.CurrentRound = 0
	r.State = StatePlaying

	log.Printf("Game started in room %s with %d rounds", r.ID, totalRounds)

	r.Broadcast <- Message{
		Type: MsgTypeGameStarted,
		Payload: map[string]interface{}{
			"total_rounds": totalRounds,
			"players":      r.getPlayerInfoList(),
		},
	}

	// Start first round after 2 seconds
	go func() {
		time.Sleep(2 * time.Second)
		r.startNextRound()
	}()
}

func (r *GameRoom) startNextRound() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.CurrentRound++
	r.Guesses = make(map[string]Guess)

	// Select track
	track := r.selectTrack()
	if track == nil {
		r.Broadcast <- Message{
			Type: MsgTypeError,
			Payload: map[string]interface{}{
				"message": "No tracks available",
			},
		}
		return
	}

	r.CurrentTrack = track

	log.Printf("Round %d/%d started in room %s - Track: %s", r.CurrentRound, r.TotalRounds, r.ID, track.Name)

	r.Broadcast <- Message{
		Type: MsgTypeRoundStarted,
		Payload: map[string]interface{}{
			"round":        r.CurrentRound,
			"total_rounds": r.TotalRounds,
			"track":        track,
			"players":      r.getPlayerInfoList(),
		},
	}

	// Set timer for 30 seconds
	if r.RoundTimer != nil {
		r.RoundTimer.Stop()
	}
	r.RoundTimer = time.AfterFunc(30*time.Second, func() {
		r.endRound()
	})
}

func (r *GameRoom) handleGuess(guess Guess) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StatePlaying {
		return
	}

	// Store guess
	r.Guesses[guess.PlayerID] = guess

	log.Printf("Player %s guessed %s in room %s", guess.PlayerID, guess.GuessedPlayerID, r.ID)

	// Broadcast guess received
	r.Broadcast <- Message{
		Type: MsgTypeGuessReceived,
		Payload: map[string]interface{}{
			"player_id":     guess.PlayerID,
			"guesses_count": len(r.Guesses),
			"total_players": len(r.Players),
		},
	}

	// End round early if all players guessed
	if len(r.Guesses) == len(r.Players) {
		if r.RoundTimer != nil {
			r.RoundTimer.Stop()
		}
		go r.endRound()
	}
}

func (r *GameRoom) endRound() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.State != StatePlaying {
		return
	}

	result := r.calculateRoundResults()

	log.Printf("Round %d complete in room %s - Winner: %s", r.CurrentRound, r.ID, result.WinnerID)

	r.Broadcast <- Message{
		Type:    MsgTypeRoundComplete,
		Payload: result,
	}

	// Check if game is over
	if r.CurrentRound >= r.TotalRounds {
		r.State = StateGameOver
		
		winnerID := r.getWinnerID()
		log.Printf("Game over in room %s - Winner: %s", r.ID, winnerID)

		r.Broadcast <- Message{
			Type: MsgTypeGameOver,
			Payload: map[string]interface{}{
				"winner_id":    winnerID,
				"final_scores": r.Scores,
				"players":      r.getPlayerInfoList(),
			},
		}

		// Reset room to waiting state after 10 seconds
		go func() {
			time.Sleep(10 * time.Second)
			r.mu.Lock()
			r.State = StateWaiting
			r.CurrentRound = 0
			r.Scores = make(map[string]int)
			for pid := range r.Players {
				r.Scores[pid] = 0
			}
			r.mu.Unlock()
		}()
	} else {
		// Start next round after 5 seconds
		go func() {
			time.Sleep(5 * time.Second)
			r.startNextRound()
		}()
	}
}

func (r *GameRoom) selectTrack() *auth.Track {
	// Build map of all tracks
	trackCounts := make(map[string]int)
	trackMap := make(map[string]*auth.Track)

	for _, player := range r.Players {
		for _, track := range player.TopTracks {
			trackCounts[track.ID]++
			if _, exists := trackMap[track.ID]; !exists {
				t := track
				trackMap[track.ID] = &t
			}
		}
	}

	// Prefer tracks that appear in multiple players' libraries
	commonTracks := make([]string, 0)
	for trackID, count := range trackCounts {
		if count >= 2 {
			commonTracks = append(commonTracks, trackID)
		}
	}

	// Fall back to all tracks if no common ones
	if len(commonTracks) == 0 {
		for trackID := range trackMap {
			commonTracks = append(commonTracks, trackID)
		}
	}

	if len(commonTracks) == 0 {
		return nil
	}

	// Select random track
	selectedID := commonTracks[rand.Intn(len(commonTracks))]
	return trackMap[selectedID]
}

func (r *GameRoom) calculateRoundResults() *RoundResult {
	// Find all rankings
	allRankings := make(map[string]int)
	for playerID, player := range r.Players {
		rank := 999 // Default rank if track not found
		for _, track := range player.TopTracks {
			if track.ID == r.CurrentTrack.ID {
				rank = track.Rank
				break
			}
		}
		allRankings[playerID] = rank
	}

	// Find winner (lowest rank)
	winnerID := ""
	bestRank := 999
	for playerID, rank := range allRankings {
		if rank < bestRank {
			bestRank = rank
			winnerID = playerID
		}
	}

	// Find correct guessers
	correctGuessers := make([]string, 0)
	for playerID, guess := range r.Guesses {
		if guess.GuessedPlayerID == winnerID {
			correctGuessers = append(correctGuessers, playerID)
		}
	}

	// Sort by timestamp (fastest first)
	sort.Slice(correctGuessers, func(i, j int) bool {
		return r.Guesses[correctGuessers[i]].Timestamp.Before(
			r.Guesses[correctGuessers[j]].Timestamp,
		)
	})

	// Award points
	pointsAwarded := make(map[string]int)
	for idx, playerID := range correctGuessers {
		basePoints := 10
		speedBonus := 0
		if idx == 0 {
			speedBonus = 5
		}

		total := basePoints + speedBonus
		pointsAwarded[playerID] = total
		r.Scores[playerID] += total
	}

	return &RoundResult{
		Round:           r.CurrentRound,
		Track:           *r.CurrentTrack,
		WinnerID:        winnerID,
		WinnerRank:      bestRank,
		CorrectGuessers: correctGuessers,
		PointsAwarded:   pointsAwarded,
		AllRankings:     allRankings,
		UpdatedScores:   r.Scores,
	}
}

func (r *GameRoom) getWinnerID() string {
	maxScore := -1
	winnerID := ""
	for playerID, score := range r.Scores {
		if score > maxScore {
			maxScore = score
			winnerID = playerID
		}
	}
	return winnerID
}

func (r *GameRoom) getPlayerInfoList() []PlayerInfo {
	players := make([]PlayerInfo, 0, len(r.PlayerOrder))
	for _, id := range r.PlayerOrder {
		if player, exists := r.Players[id]; exists {
			players = append(players, PlayerInfo{
				ID:    player.ID,
				Name:  player.Name,
				Score: r.Scores[player.ID],
			})
		}
	}
	return players
}

func (r *GameRoom) broadcastToAll(msg Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, player := range r.Players {
		if player.Connection != nil {
			ctx := context.Background()
			err := wsjson.Write(ctx, player.Connection, msg)
			if err != nil {
				log.Printf("Error broadcasting to player %s: %v", player.ID, err)
			}
		}
	}
}