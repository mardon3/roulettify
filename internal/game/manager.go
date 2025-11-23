package game

import (
	"fmt"
	"log"
	"sync"
)

type RoomManager struct {
	rooms map[string]*GameRoom
	mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
	rm := &RoomManager{
		rooms: make(map[string]*GameRoom),
	}
	
	// Initialize 3 persistent rooms
	rm.initializePersistentRooms()
	
	return rm
}

// initializePersistentRooms creates the 3 permanent game rooms
func (rm *RoomManager) initializePersistentRooms() {
	roomNames := []string{"Room 1", "Room 2", "Room 3"}
	
	for _, roomName := range roomNames {
		room := NewGameRoom(roomName)
		rm.rooms[roomName] = room
		go room.Run()
		log.Printf("Initialized persistent room: %s", roomName)
	}
}

// GetRoom returns a room by ID
func (rm *RoomManager) GetRoom(roomID string) (*GameRoom, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if room, exists := rm.rooms[roomID]; exists {
		return room, nil
	}

	return nil, fmt.Errorf("room not found - valid rooms are: Room 1, Room 2, Room 3")
}

// ListRooms returns all persistent rooms with their player counts
// Rooms are always returned in order: Room 1, Room 2, Room 3
func (rm *RoomManager) ListRooms() []RoomInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return rooms in consistent order
	roomOrder := []string{"Room 1", "Room 2", "Room 3"}
	roomInfos := make([]RoomInfo, 0, 3)
	
	for _, roomID := range roomOrder {
		if room, exists := rm.rooms[roomID]; exists {
			room.mu.RLock()
			roomInfos = append(roomInfos, RoomInfo{
				ID:          roomID,
				PlayerCount: len(room.Players),
				MaxPlayers:  MaxPlayersPerRoom,
				State:       room.State,
			})
			room.mu.RUnlock()
		}
	}
	return roomInfos
}

type RoomInfo struct {
	ID          string    `json:"id"`
	PlayerCount int       `json:"player_count"`
	MaxPlayers  int       `json:"max_players"`
	State       GameState `json:"state"`
}

func (rm *RoomManager) GetMetrics() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	totalPlayers := 0
	activePlayers := 0
	
	for _, room := range rm.rooms {
		room.mu.RLock()
		totalPlayers += len(room.Players)
		if room.State == StatePlaying {
			activePlayers += len(room.Players)
		}
		room.mu.RUnlock()
	}

	return map[string]interface{}{
		"total_rooms":    len(rm.rooms),
		"total_players":  totalPlayers,
		"active_players": activePlayers,
	}
}

