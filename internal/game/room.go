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

const MaxPlayersPerRoom = 10

type GameRoom struct {
	ID           string
	Players      map[string]*Player
	PlayerOrder  []string
	Scores       map[string]int
	CurrentRound int
	TotalRounds  int
	CurrentTrack *auth.Track
	Guesses      map[string]Guess
	PlayedTracks map[string]bool
	State        GameState
	RoundTimer   *time.Timer
	LeaderID     string
	RoundStartTime time.Time

	// Channels
	Join      chan *Player
	Leave     chan string
	Ready     chan ReadyPayload
	Guess     chan Guess
	StartGame chan StartGamePayload
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
		PlayedTracks: make(map[string]bool),
		State:        StateWaiting,
		Join:         make(chan *Player, 10),
		Leave:        make(chan string, 10),
		Ready:        make(chan ReadyPayload, 10),
		Guess:        make(chan Guess, 10),
		StartGame:    make(chan StartGamePayload, 1),
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

		case payload := <-r.Ready:
			r.handlePlayerReady(payload)

		case payload := <-r.StartGame:
			r.handleGameStart(payload)

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
				"message": "Room is full (maximum 10 players)",
			},
		}
		return
	}

	// Add player
	player.IsReady = false
	player.IsLeader = false
	
	// Assign leader if room is empty
	if len(r.Players) == 0 {
		player.IsLeader = true
		r.LeaderID = player.ID
		log.Printf("Player %s assigned as leader of room %s", player.Name, r.ID)
	}

	r.Players[player.ID] = player
	r.PlayerOrder = append(r.PlayerOrder, player.ID)
	r.Scores[player.ID] = 0

	log.Printf("Player %s joined room %s", player.Name, r.ID)

	// Broadcast player joined
	r.Broadcast <- Message{
		Type: MsgTypePlayerJoined,
		Payload: map[string]interface{}{
			"player": PlayerInfo{
				ID:       player.ID,
				Name:     player.Name,
				Score:    0,
				IsLeader: player.IsLeader,
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

	// Reassign leader if needed
	if playerID == r.LeaderID && len(r.PlayerOrder) > 0 {
		newLeaderID := r.PlayerOrder[0]
		r.LeaderID = newLeaderID
		if p, ok := r.Players[newLeaderID]; ok {
			p.IsLeader = true
			log.Printf("Player %s is now the leader of room %s", p.Name, r.ID)
		}
	} else if len(r.PlayerOrder) == 0 {
		r.LeaderID = ""
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

func (r *GameRoom) handlePlayerReady(payload ReadyPayload) {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.Players[payload.PlayerID]
	if !exists {
		return
	}

	// Check if we need to reset the game from Game Over state
	if r.State == StateGameOver {
		r.State = StateWaiting
		r.CurrentRound = 0
		r.Scores = make(map[string]int)
		for pid := range r.Players {
			r.Scores[pid] = 0
			if p, ok := r.Players[pid]; ok {
				p.IsReady = false
			}
		}

		log.Printf("Room %s reset to waiting state by player %s", r.ID, player.Name)

		r.Broadcast <- Message{
			Type: MsgTypeGameReset,
			Payload: map[string]interface{}{
				"players": r.getPlayerInfoList(),
			},
		}
	}

	player.IsReady = payload.IsReady
	log.Printf("Player %s is ready: %v", player.Name, payload.IsReady)

	r.Broadcast <- Message{
		Type: MsgTypePlayerReady,
		Payload: map[string]interface{}{
			"player_id": payload.PlayerID,
			"is_ready":  payload.IsReady,
		},
	}
}

func (r *GameRoom) handleGameStart(payload StartGamePayload) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Auto-fix state if we are stuck in GameOver but trying to start
	if r.State == StateGameOver {
		r.State = StateWaiting
		r.CurrentRound = 0
		r.Scores = make(map[string]int)
		for pid := range r.Players {
			r.Scores[pid] = 0
		}
	}

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

	// Check if all players are ready
	for _, p := range r.Players {
		if !p.IsReady {
			r.Broadcast <- Message{
				Type: MsgTypeError,
				Payload: map[string]interface{}{
					"message": "All players must be ready to start",
				},
			}
			return
		}
	}

	r.TotalRounds = payload.TotalRounds
	if r.TotalRounds <= 0 {
		r.TotalRounds = 10 // Default
	}
	
	r.CurrentRound = 0
	r.State = StatePlaying
	r.PlayedTracks = make(map[string]bool) // Reset played tracks

	log.Printf("Game started in room %s with %d rounds", 
		r.ID, payload.TotalRounds)

	r.Broadcast <- Message{
		Type: MsgTypeGameStarted,
		Payload: map[string]interface{}{
			"total_rounds": payload.TotalRounds,
			"players":      r.getPlayerInfoList(),
		},
	}

	// Start first round after 5 seconds (intermission)
	go func() {
		time.Sleep(5 * time.Second)
		r.startNextRound()
	}()
}

func (r *GameRoom) startNextRound() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.CurrentRound++
	r.RoundStartTime = time.Now()
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
	r.PlayedTracks[track.ID] = true

	log.Printf("Round %d/%d started in room %s - Track: %s", r.CurrentRound, r.TotalRounds, r.ID, track.Name)

	broadcastTrack := *track
	broadcastTrack.Name = "???"
	broadcastTrack.Artists = []string{"???"}
	broadcastTrack.ImageURL = "" // Hide album art
	// Keep PreviewURL and ID

	r.Broadcast <- Message{
		Type: MsgTypeRoundStarted,
		Payload: map[string]interface{}{
			"round":        r.CurrentRound,
			"total_rounds": r.TotalRounds,
			"track":        broadcastTrack,
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
		// Wait 5 seconds before showing game over screen
		go func() {
			time.Sleep(5 * time.Second)
			r.mu.Lock()
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
			// Skip if already played
			if r.PlayedTracks[track.ID] {
				continue
			}
			trackCounts[track.ID]++
			if _, exists := trackMap[track.ID]; !exists {
				t := track
				trackMap[track.ID] = &t
			}
		}
	}

	// Weighted selection: tracks appearing for multiple users get higher weight
	// Create a pool where tracks are added 'count' times (or count^2 for more weight)
	weightedPool := make([]string, 0)
	
	for trackID, count := range trackCounts {
		// Base weight is 1
		weight := 1
		// If track appears for multiple users, increase weight significantly
		if count > 1 {
			weight = count * 5 // Give 5x weight per occurrence if shared
		}
		
		for i := 0; i < weight; i++ {
			weightedPool = append(weightedPool, trackID)
		}
	}

	if len(weightedPool) == 0 {
		return nil
	}

	// Select random track from weighted pool
	selectedID := weightedPool[rand.Intn(len(weightedPool))]
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

	// Award points and calculate durations
	pointsAwarded := make(map[string]int)
	guessDurations := make(map[string]float64)
	
	for idx, playerID := range correctGuessers {
		basePoints := 10
		speedBonus := 0
		if idx == 0 {
			speedBonus = 5
		}

		total := basePoints + speedBonus
		pointsAwarded[playerID] = total
		r.Scores[playerID] += total
		
		// Calculate duration
		duration := r.Guesses[playerID].Timestamp.Sub(r.RoundStartTime).Seconds()
		guessDurations[playerID] = duration
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
		GuessDurations:  guessDurations,
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
				ID:       player.ID,
				Name:     player.Name,
				Score:    r.Scores[player.ID],
				IsReady:  player.IsReady,
				IsLeader: player.IsLeader,
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