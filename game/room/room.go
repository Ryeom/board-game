package room

import (
	"github.com/Ryeom/board-game/session"
	"time"
)

type GameMode string

const (
	GameModeHanabi GameMode = "hanabi"
)

type Room struct {
	ID        string
	Host      *Attender
	Players   []*Attender
	GameMode  GameMode
	Engine    GameEngine
	State     any
	CreatedAt time.Time
}

var rooms = map[string]*Room{}

func CreateRoom(roomID string, user *session.UserSession) *Room {
	host := NewAttender(user.ID, user.Name, true)
	r := &Room{
		ID:        roomID,
		Host:      host,
		Players:   []*Attender{host},
		GameMode:  GameModeHanabi,
		CreatedAt: time.Now(),
	}
	rooms[roomID] = r
	return r
}

func JoinRoom(roomID string, user *session.UserSession) *Room {
	if r, ok := rooms[roomID]; ok {
		r.Players = append(r.Players, NewAttender(user.ID, user.Name, false))
		return r
	}
	return nil
}

func GetRoom(roomID string) (*Room, bool) {
	r, ok := rooms[roomID]
	return r, ok
}
