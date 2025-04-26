package redisconn

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
)

var (
	RoomClient *redis.Client
)

func CreateClient(addr, pw string, db int) (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pw,
		DB:       db,
	})
	ctx := context.Background()
	if err := c.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis 연결 실패 %s [%d] : %v", addr, db, err)
		return nil, err
	}
	log.Println("✅ Redis 연결 성공")

	return c, nil
}
