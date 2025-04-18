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
