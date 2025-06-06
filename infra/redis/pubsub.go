package redisutil

import (
	"context"
	"github.com/Ryeom/board-game/log"
	"github.com/redis/go-redis/v9"
)

// PubSubSubscribe 채널 구독
func PubSubSubscribe(target string, channel string) *redis.PubSub {
	rdb := Client[target]
	if rdb == nil {
		return nil
	}
	ctx := context.Background()
	return rdb.Subscribe(ctx, channel)
}

// PubSubPublish 메시지 발행
func PubSubPublish(target string, channel string, message string) {
	rdb := Client[target]
	if rdb == nil {
		return
	}
	ctx := context.Background()
	err := rdb.Publish(ctx, channel, message).Err()
	if err != nil {
		log.Logger.Errorf("PubSub Publish ERROR: %v", err)
	}
}

// PubSubReceive 메시지 수신
func PubSubReceive(pubsub *redis.PubSub) *redis.Message {
	ctx := context.Background()
	msg, err := pubsub.ReceiveMessage(ctx)
	if err != nil {
		log.Logger.Errorf("PubSub Receive ERROR: %v", err)
		return nil
	}
	return msg
}

// PubSubUnsubscribe 채널 구독 해제
func PubSubUnsubscribe(pubsub *redis.PubSub, channels ...string) {
	ctx := context.Background()
	err := pubsub.Unsubscribe(ctx, channels...)
	if err != nil {
		log.Logger.Errorf("PubSub Unsubscribe ERROR: %v", err)
	}
}

// PubSubClose 종료
func PubSubClose(pubsub *redis.PubSub) {
	err := pubsub.Close()
	if err != nil {
		log.Logger.Errorf("PubSub Close ERROR: %v", err)
	}
}
