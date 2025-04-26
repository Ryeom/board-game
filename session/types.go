package session

import (
	"github.com/gorilla/websocket"
	"time"
)

type UserSession struct {
	ID          string          // 내부 식별용 소켓ID (UUID 기반)
	Name        string          // 유저 이름 (identify 이벤트에서 세팅)
	Connection  *websocket.Conn // WebSocket 연결 객체
	RoomID      string          // 참가 중인 방 ID (없으면 "")
	IsHost      bool            // 방장 여부
	ConnectedAt time.Time       // 연결된 시각 (for 세션 만료, 로깅)
	LastPingAt  time.Time       // 마지막 Ping/Pong 수신 시각 (for 연결 감시)
	IP          string          // 클라이언트 RealIP
	UserAgent   string          // 접속한 디바이스/브라우저 정보
}

func CreateUserSession(conn *websocket.Conn, socketId string) *UserSession {
	return &UserSession{
		ID:          socketId,
		Connection:  conn,
		ConnectedAt: time.Now(),
		LastPingAt:  time.Now(),
	}
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
