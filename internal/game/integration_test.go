package game

import (
	"testing"
)

// TestPersistentRoomsInitialization verifies 3 rooms are created on startup
func TestPersistentRoomsInitialization(t *testing.T) {
	manager := NewRoomManager()

	// Verify we have exactly 3 rooms
	if len(manager.rooms) != 3 {
		t.Errorf("Expected 3 persistent rooms, got %d", len(manager.rooms))
	}

	// Verify room names
	expectedRooms := []string{"Room 1", "Room 2", "Room 3"}
	for _, roomName := range expectedRooms {
		room, err := manager.GetRoom(roomName)
		if err != nil {
			t.Errorf("Expected room '%s' to exist, got error: %v", roomName, err)
		}
		if room == nil {
			t.Errorf("Room '%s' is nil", roomName)
		}
	}

	t.Logf("✓ 3 persistent rooms correctly initialized")
}

// TestGetRoomReturnsExisting verifies getting existing rooms works
func TestGetRoomReturnsExisting(t *testing.T) {
	manager := NewRoomManager()

	// Should be able to get existing rooms
	room, err := manager.GetRoom("Room 1")
	if err != nil {
		t.Fatalf("Failed to get Room 1: %v", err)
	}
	if room == nil {
		t.Fatal("Room 1 should not be nil")
	}

	// Getting same room again should return same instance
	room2, err := manager.GetRoom("Room 1")
	if err != nil {
		t.Fatalf("Failed to get Room 1 again: %v", err)
	}
	if room != room2 {
		t.Error("Should return same room instance")
	}

	t.Logf("✓ GetRoom correctly returns existing rooms")
}

// TestGetRoomRejectsInvalid verifies invalid room names are rejected
func TestGetRoomRejectsInvalid(t *testing.T) {
	manager := NewRoomManager()

	// Should fail for non-existent room
	_, err := manager.GetRoom("InvalidRoom")
	if err == nil {
		t.Error("Should reject invalid room name")
	}

	expectedErr := "room not found - valid rooms are: Room 1, Room 2, Room 3"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}

	t.Logf("✓ Invalid room names correctly rejected")
}

// TestChannelBufferSizes verifies channel buffers are correctly sized
func TestChannelBufferSizes(t *testing.T) {
	room := NewGameRoom("test-room")

	tests := []struct {
		name     string
		expected int
		actual   int
	}{
		{"Join channel", 10, cap(room.Join)},
		{"Leave channel", 10, cap(room.Leave)},
		{"Guess channel", 10, cap(room.Guess)},
		{"StartGame channel", 1, cap(room.StartGame)},
		{"Broadcast channel", 10, cap(room.Broadcast)},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("%s: expected capacity %d, got %d", tt.name, tt.expected, tt.actual)
		}
	}

	t.Logf("✓ All channel buffers correctly sized")
}

// TestRoomReuse verifies getting existing room doesn't create duplicate
func TestRoomReuse(t *testing.T) {
	manager := NewRoomManager()

	room1, err := manager.GetRoom("Room 1")
	if err != nil {
		t.Fatalf("Failed to get room: %v", err)
	}

	room2, err := manager.GetRoom("Room 1")
	if err != nil {
		t.Fatalf("Failed to get existing room: %v", err)
	}

	if room1 != room2 {
		t.Error("Should return same room instance for same ID")
	}

	// Should still have exactly 3 rooms
	if len(manager.rooms) != 3 {
		t.Errorf("Expected 3 rooms, got %d", len(manager.rooms))
	}

	t.Logf("✓ Room reuse works correctly")
}

// TestMetrics verifies metrics reporting
func TestMetrics(t *testing.T) {
	manager := NewRoomManager()

	// Get existing persistent rooms
	room1, _ := manager.GetRoom("Room 1")
	room2, _ := manager.GetRoom("Room 2")

	// Setup room1: playing with 2 players
	room1.mu.Lock()
	room1.State = StatePlaying
	room1.Players = map[string]*Player{
		"p1": nil,
		"p2": nil,
	}
	room1.mu.Unlock()

	// Setup room2: waiting with 1 player
	room2.mu.Lock()
	room2.State = StateWaiting
	room2.Players = map[string]*Player{
		"p3": nil,
	}
	room2.mu.Unlock()

	metrics := manager.GetMetrics()

	// We have 3 persistent rooms total
	if metrics["total_rooms"] != 3 {
		t.Errorf("Expected 3 total rooms, got %v", metrics["total_rooms"])
	}

	if metrics["total_players"] != 3 {
		t.Errorf("Expected 3 total players, got %v", metrics["total_players"])
	}

	if metrics["active_players"] != 2 {
		t.Errorf("Expected 2 active players (in StatePlaying), got %v", metrics["active_players"])
	}

	t.Logf("✓ Metrics reporting works correctly")
}

// TestConcurrentRoomAccess verifies thread safety
func TestConcurrentRoomAccess(t *testing.T) {
	manager := NewRoomManager()

	done := make(chan bool, 10)

	// Access rooms concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			roomID := "Room " + string(rune('1'+(id%3)))
			_, err := manager.GetRoom(roomID)
			if err != nil {
				t.Errorf("Failed to get room %s: %v", roomID, err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still have exactly 3 rooms
	if len(manager.rooms) != 3 {
		t.Errorf("Expected 3 rooms, got %d", len(manager.rooms))
	}

	t.Logf("✓ Concurrent room access is thread-safe")
}
