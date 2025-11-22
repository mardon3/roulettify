package game

import (
	"sync"
)

type RoomManager struct {
	rooms map[string]*GameRoom
	mu    sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*GameRoom),
	}
}

func (rm *RoomManager) CreateRoom(roomID string) (*GameRoom, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if room already exists
	if room, exists := rm.rooms[roomID]; exists {
		return room, nil
	}

	// Create new room
	room := NewGameRoom(roomID)
	rm.rooms[roomID] = room

	// Start room goroutine
	go room.Run()

	return room, nil
}

func (rm *RoomManager) GetRoom(roomID string) (*GameRoom, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	room, exists := rm.rooms[roomID]
	return room, exists
}

func (rm *RoomManager) DeleteRoom(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if room, exists := rm.rooms[roomID]; exists {
		close(room.Shutdown)
		delete(rm.rooms, roomID)
	}
}

func (rm *RoomManager) ListRooms() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	rooms := make([]string, 0, len(rm.rooms))
	for roomID := range rm.rooms {
		rooms = append(rooms, roomID)
	}
	return rooms
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