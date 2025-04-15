package room

import (
	"sync"
	"time"
)

type GameMode string

const (
	GameModeHanabi GameMode = "hanabi"
	GameModeUno    GameMode = "uno"
)

type Room struct {
	ID        string
	Players   []*Attender
	GameMode  GameMode
	Engine    GameEngine
	State     any
	CreatedAt time.Time
}

type RoomManager struct {
	rooms map[string]*Room
	lock  sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

func (m *RoomManager) CreateRoom(id string, host *Attender, mode GameMode, engine GameEngine) *Room {
	room := &Room{
		ID:        id,
		Players:   []*Attender{host},
		GameMode:  mode,
		Engine:    engine,
		CreatedAt: time.Now(),
	}
	m.lock.Lock()
	m.rooms[id] = room
	m.lock.Unlock()
	return room
}

func (m *RoomManager) GetRoom(id string) (*Room, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	r, ok := m.rooms[id]
	return r, ok
}

func (m *RoomManager) DeleteRoom(id string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.rooms, id)
}

func (m *RoomManager) ListRooms() []*Room {
	m.lock.RLock()
	defer m.lock.RUnlock()

	list := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		list = append(list, r)
	}
	return list
}
