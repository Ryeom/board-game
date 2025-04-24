package redisconn

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"log"
)

var (
	Client *redis.Client
)

func Initialize() {
	addr := viper.GetString("redis.addr")
	Client = newClient(addr, "", viper.GetInt("redis.room-index"))
}

func newClient(addr, pw string, db int) *redis.Client {
	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pw,
		DB:       db,
	})
	ctx := context.Background()
	if err := c.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Redis 연결 실패 %s [%d] : %v", addr, db, err)
	} else {
		log.Println("✅ Redis 연결 성공")
	}
	return c
}
