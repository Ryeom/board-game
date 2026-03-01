package user

import (
	"errors"
	"fmt"
	"github.com/Ryeom/board-game/log"
	"sync"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/gorilla/websocket"
)

const sessionTTL = 3 * time.Hour // 세션 TTL
type Session struct {
	ID           string          `json:"id"`
	ActualUserID string          `json:"actualUserId"`
	Name         string          `json:"name"`
	RoomID       string          `json:"roomId"`
	IsHost       bool            `json:"isHost"`
	ConnectedAt  time.Time       `json:"connectedAt"`
	LastPingAt   time.Time       `json:"lastPingAt"`
	IP           string          `json:"ip"`
	UserAgent    string          `json:"userAgent"`
	Status       string          `json:"status"`
	Conn         *websocket.Conn `json:"-"`
	WriteMutex   *sync.Mutex     `json:"-"`
}

func NewUserSession(socketID, name, roomID, ip, userAgent string, isHost bool, conn *websocket.Conn) *Session {
	now := time.Now()
	return &Session{
		ID:           socketID,
		ActualUserID: "",
		Name:         name,
		RoomID:       roomID,
		IsHost:       isHost,
		ConnectedAt:  now,
		LastPingAt:   now,
		IP:           ip,
		UserAgent:    userAgent,
		Status:       "connected",
		Conn:         conn,
		WriteMutex:   &sync.Mutex{},
	}
}

// SaveUserSession 사용자 세션을 Redis에 저장
func SaveUserSession(session *Session) error {
	return redisutil.SaveJSON(redisutil.RedisTargetUser, sessionKey(session.ID), session, sessionTTL)
}

// GetSession socketID로 사용자 세션을 조회
func GetSession(socketID string) (*Session, error) {
	var session Session
	found := redisutil.GetJSON(redisutil.RedisTargetUser, sessionKey(socketID), &session)
	if !found {
		return nil, errors.New("session not found or an error occurred")
	}
	session.WriteMutex = &sync.Mutex{}
	return &session, nil
}

func DeleteUserSession(socketID string) error {
	return redisutil.Delete(redisutil.RedisTargetUser, sessionKey(socketID))
}

// GetSessionsByRoom 주어진 roomID에 속한 모든 사용자 세션 조회
func GetSessionsByRoom(roomID string) ([]*Session, error) {
	socketIDs, err := redisutil.GetSetMembers(redisutil.RedisTargetUser, RoomIndexKey(roomID))
	if err != nil {
		return nil, fmt.Errorf("failed to get room session members for room %s: %w", roomID, err)
	}

	var sessions []*Session
	for _, socketID := range socketIDs {
		session, err := GetSession(socketID)
		if err != nil {
			log.Logger.Warningf("GetSessionsByRoom - Failed to get session %s: %v", socketID, err)
			_ = redisutil.RemoveSetMembers(redisutil.RedisTargetUser, RoomIndexKey(roomID), socketID)
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func sessionKey(socketID string) string {
	return fmt.Sprintf("user:session:%s", socketID)
}

func RoomIndexKey(roomID string) string {
	return fmt.Sprintf("room_sessions:%s", roomID)
}
