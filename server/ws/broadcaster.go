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
}

var GlobalBroadcaster Broadcaster

type RedisBroadcaster struct {
	pubsub *redis.PubSub
	ctx    context.Context
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
func BroadcastToRoom() {

}
func BroadcastToAllUser() {

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

		sessions, err := user.GetSessionsByRoom(parsed.RoomID)
		if err != nil {
			log.Println("❌ Redis 세션 조회 실패:", err)
			continue
		}

		for _, s := range sessions {
			if s.Conn != nil {
				s.Conn.WriteJSON(parsed.Data)
			}
		}
	}
}
