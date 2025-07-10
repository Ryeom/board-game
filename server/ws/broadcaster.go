package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Ryeom/board-game/internal/user"
	"log"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/redis/go-redis/v9"
)

type Broadcaster interface {
	BroadcastToRoom(roomID string, payload any) error
	SetSessionGetter(getter func(socketID string) (*user.Session, bool))
}

var GlobalBroadcaster Broadcaster

type RedisBroadcaster struct {
	pubsub        *redis.PubSub
	ctx           context.Context
	sessionGetter func(socketID string) (*user.Session, bool)
}

func NewRedisBroadcaster(ctx context.Context) *RedisBroadcaster {
	channel := "broadcast:room"
	pubsub := redisutil.Client[redisutil.RedisTargetPubSub].Subscribe(ctx, channel)

	err := pubsub.Ping(ctx)
	if err != nil {
		log.Fatal(err)
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

func (rb *RedisBroadcaster) BroadcastToRoom(roomID string, payload any) error {
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

func (rb *RedisBroadcaster) listen() {
	for msg := range rb.pubsub.Channel() {
		var parsed struct {
			RoomID string      `json:"roomId"`
			Data   interface{} `json:"data"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &parsed); err != nil {
			log.Println("❌ Redis 메시지 파싱 실패:", err)
			continue
		}

		sessionIDsInRoom, err := redisutil.GetSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(parsed.RoomID))
		if err != nil {
			log.Println("❌ Redis 세션 ID 조회 실패:", err)
			continue
		}

		for _, sID := range sessionIDsInRoom {
			if rb.sessionGetter == nil {
				log.Println("❌ Broadcaster session getter not set. Cannot broadcast to live sessions.")
				continue
			}
			liveSession, found := rb.sessionGetter(sID)
			if found && liveSession.Conn != nil {
				err := liveSession.Conn.WriteJSON(parsed.Data)
				if err != nil {
					log.Println("❌ Failed to write JSON to WebSocket for session", sID, ":", err)
				}
			} else {
				log.Printf("Session %s not found in active connections or connection is nil. Room: %s. Cleaning up stale Redis entry.", sID, parsed.RoomID)
				_ = redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(parsed.RoomID), sID)
			}
		}
	}
}
