// session/types.go
package session

import "github.com/gorilla/websocket"

type UserSession struct {
	ID         string
	Name       string
	Connection *websocket.Conn
	RoomID     string
	IsHost     bool
}
