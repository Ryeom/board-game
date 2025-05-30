package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisBroadcaster struct {
	Client *redis.Client
}

func NewRedisBroadcaster(client *redis.Client) *RedisBroadcaster {
	return &RedisBroadcaster{Client: client}
}

func (b *RedisBroadcaster) BroadcastToRoom(roomID string, payload any) {
	msg, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("🔴 Failed to marshal broadcast payload:", err)
		return
	}

	channel := fmt.Sprintf("room:%s", roomID)
	if err := b.Client.Publish(context.Background(), channel, msg).Err(); err != nil {
		fmt.Println("🔴 Redis publish failed:", err)
	}
}
