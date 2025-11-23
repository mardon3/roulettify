package game

import (
	"testing"
	"time"

	"roulettify/internal/auth"
)

// TestRoomCapacityLimit verifies the 6 player limit per room
func TestRoomCapacityLimit(t *testing.T) {
	room := NewGameRoom("test-room")
	
	// Start room goroutine
	go room.Run()
	
	// Add 6 players (should succeed)
	for i := 0; i < MaxPlayersPerRoom; i++ {
		player := &Player{
			Player: &auth.Player{
				ID:        string(rune('A' + i)),
				Name:      "Player " + string(rune('A'+i)),
				SpotifyID: "spotify-" + string(rune('A'+i)),
				TopTracks: make([]auth.Track, 0),
			},
			Connection: nil,
			JoinedAt:   time.Now(),
		}
		
		room.Join <- player
		time.Sleep(10 * time.Millisecond) // Let handler process
	}
	
	// Verify we have exactly 6 players
	room.mu.RLock()
	playerCount := len(room.Players)
	room.mu.RUnlock()
	
	if playerCount != MaxPlayersPerRoom {
		t.Errorf("Expected %d players, got %d", MaxPlayersPerRoom, playerCount)
	}
	
	// Try to add 7th player (should be rejected)
	player7 := &Player{
		Player: &auth.Player{
			ID:        "player7",
			Name:      "Player 7",
			SpotifyID: "spotify-7",
			TopTracks: make([]auth.Track, 0),
		},
		Connection: nil,
		JoinedAt:   time.Now(),
	}
	
	room.Join <- player7
	time.Sleep(50 * time.Millisecond) // Let handler process
	
	// Verify still only 6 players
	room.mu.RLock()
	finalCount := len(room.Players)
	room.mu.RUnlock()
	
	if finalCount != MaxPlayersPerRoom {
		t.Errorf("Expected %d players after reject, got %d", MaxPlayersPerRoom, finalCount)
	}
	
	// Verify player7 was not added
	room.mu.RLock()
	_, exists := room.Players["player7"]
	room.mu.RUnlock()
	
	if exists {
		t.Error("Player 7 should not have been added (room at capacity)")
	}
	
	t.Logf("✓ Room correctly enforces %d player limit", MaxPlayersPerRoom)
}
