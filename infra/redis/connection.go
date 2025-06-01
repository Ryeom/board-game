package redisutil

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"log"
)

var (
	RoomClient *redis.Client
	UserClient *redis.Client
)

func Initialize() {
	var err error
	RoomClient, err = createClient(viper.GetString("redis.addr"), "", 0)
	if err != nil {
		panic(err)
	}
	UserClient, err = createClient(viper.GetString("redis.addr"), "", 0)
	if err != nil {
		panic(err)
	}
}
func createClient(addr, pw string, db int) (*redis.Client, error) {
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
