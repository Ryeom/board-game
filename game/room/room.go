// room/room.go
package room

import "time"

type GameMode string

const (
	GameModeHanabi GameMode = "hanabi"
)

type Room struct {
	ID        string
	Host      Player
	Players   []Player
	GameMode  GameMode
	Engine    GameEngine
	State     any
	CreatedAt time.Time
}

var rooms = map[string]*Room{}

func CreateRoom(roomID string, host Player) *Room {
	r := &Room{
		ID:        roomID,
		Host:      host,
		Players:   []Player{host},
		GameMode:  GameModeHanabi,
		CreatedAt: time.Now(),
	}
	rooms[roomID] = r
	return r
}

func JoinRoom(roomID string, user Player) *Room {
	if r, ok := rooms[roomID]; ok {
		r.Players = append(r.Players, user)
		return r
	}
	return nil
}

func GetRoom(roomID string) (*Room, bool) {
	r, ok := rooms[roomID]
	return r, ok
}
