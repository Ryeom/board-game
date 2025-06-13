package user

import (
	"errors" // errors 패키지 임포트
	"fmt"
	"github.com/Ryeom/board-game/log"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/gorilla/websocket"
)

const sessionTTL = 3 * time.Hour // 세션 TTL
type Session struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	RoomID      string          `json:"roomId"`
	IsHost      bool            `json:"isHost"`
	ConnectedAt time.Time       `json:"connectedAt"`
	LastPingAt  time.Time       `json:"lastPingAt"`
	IP          string          `json:"ip"`
	UserAgent   string          `json:"userAgent"`
	Status      string          `json:"status"`
	Conn        *websocket.Conn `json:"-"`
}

func NewUserSession(socketID, name, roomID, ip, userAgent string, isHost bool, conn *websocket.Conn) *Session {
	now := time.Now()
	return &Session{
		ID:          socketID,
		Name:        name,
		RoomID:      roomID,
		IsHost:      isHost,
		ConnectedAt: now,
		LastPingAt:  now,
		IP:          ip,
		UserAgent:   userAgent,
		Status:      "connected",
		// Conn:        conn,
	}
}

// SaveUserSession 사용자 세션을 Redis에 저장
func SaveUserSession(session *Session) error {
	redisutil.SaveJSON(redisutil.RedisTargetUser, sessionKey(session.ID), session, sessionTTL) //
	return nil
}

// GetSession socketID로 사용자 세션을 조회
func GetSession(socketID string) (*Session, error) {
	var session Session
	found := redisutil.GetJSON(redisutil.RedisTargetUser, sessionKey(socketID), &session) //
	if !found {
		return nil, errors.New("session not found or an error occurred")
	}
	return &session, nil
}

func DeleteUserSession(socketID string) error {
	return redisutil.Delete(redisutil.RedisTargetUser, sessionKey(socketID))
}

// GetSessionsByRoom 주어진 roomID에 속한 모든 사용자 세션 조회
func GetSessionsByRoom(roomID string) ([]*Session, error) {
	// 1. 해당 방의 세션 ID Set에서 모든 멤버(socketID) 조회
	socketIDs, err := redisutil.GetSetMembers(redisutil.RedisTargetUser, RoomIndexKey(roomID))
	if err != nil {
		return nil, fmt.Errorf("failed to get room session members for room %s: %w", roomID, err)
	}

	var sessions []*Session
	for _, socketID := range socketIDs {
		session, err := GetSession(socketID) // 각 socketID에 해당하는 세션 객체를 조회
		if err != nil {
			// 세션을 찾을 수 없거나 에러가 발생하면 로그를 남기고 다음 세션으로 넘어갑니다.
			// (예: 세션 만료로 Redis에서 삭제되었지만 Set에서는 아직 제거되지 않은 경우)
			log.Logger.Warningf("GetSessionsByRoom - Failed to get session %s: %v", socketID, err)
			// 불일치 제거를 위해 Set에서 해당 멤버를 제거하는 로직을 여기에 추가할 수 있습니다.
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
