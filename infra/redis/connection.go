package redisutil

import (
	"context"
	"fmt"
	"github.com/Ryeom/board-game/log"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

const (
	RedisTargetRoom   = "room"
	RedisTargetUser   = "user"
	RedisTargetPubSub = "pubSub"
	RedisTargetGame   = "game"
)

var Client map[string]*redis.Client

func Initialize() {
	Client = map[string]*redis.Client{}
	userClient, err := CreateClient(viper.GetInt("redis.user-index"))
	if err != nil {
		log.Logger.Fatal(err)
		panic(err)
	}
	Client[RedisTargetUser] = userClient
	roomClient, err := CreateClient(viper.GetInt("redis.room-index"))
	if err != nil {
		log.Logger.Fatal(err)
		panic(err)
	}
	Client[RedisTargetRoom] = roomClient
	pubSubClient, err := CreateClient(viper.GetInt("redis.queue-index"))
	if err != nil {
		log.Logger.Fatal(err)
		panic(err)
	}
	Client[RedisTargetPubSub] = pubSubClient
	gameClient, err := CreateClient(viper.GetInt("redis.game-index"))
	if err != nil {
		log.Logger.Fatal(err)
		panic(err)
	}
	Client[RedisTargetGame] = gameClient
}
func CreateClient(db int) (*redis.Client, error) {
	addr := viper.GetString("redis.addr")
	pw := viper.GetString("redis.pw")
	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pw,
		DB:       db,
	})
	ctx := context.Background()
	if err := c.Ping(ctx).Err(); err != nil {
		log.Logger.Fatal("❌ Redis 연결 실패 %s [%d] : %v", addr, db, err)
		return nil, err
	}
	fmt.Println("✅ Redis 연결 성공")
	return c, nil
}
