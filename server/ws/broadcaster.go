package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/redis/go-redis/v9"
)

type Broadcaster interface {
	BroadcastToRoom(roomID string, payload interface{}) error
	SendToPlayer(playerID string, payload interface{}) error
	SetSessionGetter(getter func(socketID string) (*user.Session, bool))
}

var GlobalBroadcaster Broadcaster

type RedisBroadcaster struct {
	pubsub        *redis.PubSub
	ctx           context.Context
	sessionGetter func(socketID string) (*user.Session, bool)
}

func NewRedisBroadcaster(ctx context.Context) *RedisBroadcaster {
	channel := "broadcast:room" // 방 전체 메시지를 위한 채널
	pubsub := redisutil.Client[redisutil.RedisTargetPubSub].Subscribe(ctx, channel)

	err := pubsub.Ping(ctx)
	if err != nil {
		log.Logger.Error(err)
	}
	rb := &RedisBroadcaster{
		ctx:    ctx,
		pubsub: pubsub,
	}
	go rb.listen()
	return rb
}

func (rb *RedisBroadcaster) SetSessionGetter(getter func(socketID string) (*user.Session, bool)) {
	rb.sessionGetter = getter
}

func (rb *RedisBroadcaster) BroadcastToRoom(roomID string, payload interface{}) error {
	msg := map[string]any{
		"roomId": roomID,
		"data":   payload,
		"ts":     time.Now().UnixMilli(),
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("브로드캐스트 직렬화 실패: %w", err)
	}
	return redisutil.Client[redisutil.RedisTargetPubSub].Publish(rb.ctx, "broadcast:room", b).Err()
}

func (rb *RedisBroadcaster) SendToPlayer(playerID string, payload interface{}) error {
	if rb.sessionGetter == nil {
		return fmt.Errorf("Broadcaster session getter not set. Cannot send to specific player.")
	}

	liveSession, found := rb.sessionGetter(playerID)
	if !found || liveSession.Conn == nil {
		log.Logger.Errorf("Session %s not found in active connections or connection is nil. Cannot send message.", playerID)
		// TODO: 분산 환경 -> 해당 playerID의 Redis 채널로 발행하여 다른 인스턴스로 전달
		return fmt.Errorf("player session not active or not found locally")
	}

	liveSession.WriteMutex.Lock()
	defer liveSession.WriteMutex.Unlock()

	err := liveSession.Conn.WriteJSON(payload)
	if err != nil {
		log.Logger.Errorf("Failed to write JSON to WebSocket for session %s: %v", playerID, err)
		return fmt.Errorf("failed to send message to player: %w", err)
	}
	return nil
}

func (rb *RedisBroadcaster) listen() {
	for msg := range rb.pubsub.Channel() {
		var parsed struct {
			RoomID string         `json:"roomId"`
			Data   map[string]any `json:"data"`
			Ts     int64          `json:"ts"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &parsed); err != nil {
			log.Logger.Error("❌ Redis 메시지 파싱 실패:", err)
			continue
		}

		sessionIDsInRoom, err := redisutil.GetSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(parsed.RoomID))
		if err != nil {
			log.Logger.Error("❌ Redis 세션 ID 조회 실패:", err)
			continue
		}

		for _, sID := range sessionIDsInRoom {
			if rb.sessionGetter == nil {
				log.Logger.Error("❌ Broadcaster session getter not set. Cannot broadcast to live sessions.")
				continue
			}
			liveSession, found := rb.sessionGetter(sID)
			if found && liveSession.Conn != nil {
				liveSession.WriteMutex.Lock()
				err := liveSession.Conn.WriteJSON(parsed.Data)
				liveSession.WriteMutex.Unlock()
				if err != nil {
					log.Logger.Error("❌ Failed to write JSON to WebSocket for session", sID, ":", err)
				}
			} else {
				log.Logger.Error("Session %s not found in active connections or connection is nil. Room: %s. Cleaning up stale Redis entry.", sID, parsed.RoomID)
				_ = redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(parsed.RoomID), sID)
			}
		}
	}
}
