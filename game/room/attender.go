package room

import (
	"github.com/gorilla/websocket"
	"time"
)

type Attender struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	IsHost     bool            `json:"isHost"`
	JoinedAt   int64           `json:"joinedAt"`
	SocketID   string          `json:"socketId"`
	Connection *websocket.Conn `json:"-"`
	Ready      bool            `json:"ready"`
}

func NewAttender(socketID, name string, isHost bool) *Attender {
	return &Attender{
		ID:       socketID,
		Name:     name,
		IsHost:   isHost,
		JoinedAt: time.Now().Unix(),
		SocketID: socketID,
	}
}
