package room

import (
	"sync"
	"time"
)

// Player 인터페이스는 방에 참여 가능한 유저의 최소 요구 정보를 정의합니다.
type Player interface {
	GetID() string
	GetName() string
	IsHostUser() bool
}

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewRoomManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

func (rm *Manager) CreateRoom(roomID string, host Player) *Room {
	r := &Room{
		ID:        roomID,
		Host:      host,
		Players:   []Player{host},
		GameMode:  GameModeHanabi,
		CreatedAt: time.Now(),
	}
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.rooms[roomID] = r
	return r
}

func (rm *Manager) GetRoom(roomID string) (*Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	r, ok := rm.rooms[roomID]
	return r, ok
}

func (m *Manager) DeleteRoom(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, roomID)
}

func (m *Manager) ListRooms() []*Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		list = append(list, r)
	}
	return list
}
