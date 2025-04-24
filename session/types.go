package session

import "github.com/gorilla/websocket"

type UserSession struct {
	ID         string
	Name       string
	Connection *websocket.Conn
	RoomID     string
	IsHost     bool
}

type SocketEvent struct {
	Type   string         `json:"type"`
	RoomID string         `json:"roomId"`
	Name   string         `json:"name"`
	Data   map[string]any `json:"data"`
}

func (u *UserSession) GetID() string    { return u.ID }
func (u *UserSession) GetName() string  { return u.Name }
func (u *UserSession) IsHostUser() bool { return u.IsHost }

type userSessionWrapper struct {
	*UserSession
}

func (u *userSessionWrapper) GetID() string    { return u.ID }
func (u *userSessionWrapper) GetName() string  { return u.ID }
func (u *userSessionWrapper) IsHostUser() bool { return true }
