package ws

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const sessionTTL = 3 * time.Hour

type UserSession struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	RoomID      string          `json:"roomId"`
	IsHost      bool            `json:"isHost"`
	ConnectedAt time.Time       `json:"connectedAt"`
	LastPingAt  time.Time       `json:"lastPingAt"`
	IP          string          `json:"ip"`
	UserAgent   string          `json:"userAgent"`
	Conn        *websocket.Conn `json:"-"` // WebSocket 연결은 Redis에 저장하지 않음
}

func NewUserSession(socketID, name, roomID, ip, userAgent string, isHost bool, conn *websocket.Conn) *UserSession {
	now := time.Now()
	return &UserSession{
		ID:          socketID,
		Name:        name,
		RoomID:      roomID,
		IsHost:      isHost,
		ConnectedAt: now,
		LastPingAt:  now,
		IP:          ip,
		UserAgent:   userAgent,
		Conn:        conn,
	}
}

func sessionKey(socketID string) string {
	return fmt.Sprintf("session:%s", socketID)
}

func roomIndexKey(roomID string) string {
	return fmt.Sprintf("room_sessions:%s", roomID)
}

func (u *UserSession) GetID() string {
	return u.ID
}

func (u *UserSession) GetName() string {
	return u.Name
}

func (u *UserSession) IsHostUser() bool {
	return u.IsHost
}
