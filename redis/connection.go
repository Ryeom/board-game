package redisconn

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
)

var (
	Client *redis.Client
)

func Initialize() {
	addr := "127.0.0.1:6379"
	Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       1,
	})
	ctx := context.Background()
	if err := Client.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis 연결 실패: %v", err)
	} else {
		log.Println("✅ Redis 연결 성공")
	}

}
