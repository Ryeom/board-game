package room

import (
	"context"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"time"
)

type GameMode string

const (
	GameModeHanabi GameMode = "hanabi"
)

type Room struct {
	ID        string    `json:"id"`
	RoomName  string    `json:"roomName"`
	Host      string    `json:"host"`
	Players   []string  `json:"players"`
	GameMode  GameMode  `json:"gameMode"`
	CreatedAt time.Time `json:"createdAt"`
}

func CreateRoom(ctx context.Context, roomID string, hostID string) *Room {
	r := &Room{
		ID:        roomID,
		Host:      hostID,
		Players:   []string{hostID},
		GameMode:  GameModeHanabi,
		CreatedAt: time.Now(),
	}
	if err := r.Save(); err != nil {
		return nil
	}
	return r
}

func GetRoom(ctx context.Context, roomID string) (*Room, bool) {
	var r Room
	ok := redisutil.GetJSON("room", "room:"+roomID, &r)
	return &r, ok
}

func DeleteRoom(ctx context.Context, roomID string) error {
	rdb := redisutil.Client["room"]
	if rdb == nil {
		return fmt.Errorf("redis client not found")
	}
	return rdb.Del(ctx, roomID).Err()
}

func ListRooms(ctx context.Context) []*Room {
	rdb := redisutil.Client["room"]
	if rdb == nil {
		return nil
	}
	keys, err := rdb.Keys(ctx, "room:*").Result()
	if err != nil {
		return nil
	}

	var rooms []*Room
	for _, key := range keys {
		var r Room
		if ok := redisutil.GetJSON("room", key, &r); ok {
			rooms = append(rooms, &r)
		}
	}
	return rooms
}

func (r *Room) Save() error {
	redisutil.SaveJSON("room", "room:"+r.ID, r, 0)
	return nil
}

func (r *Room) Join(ctx context.Context, userID string) (bool, error) {
	for _, p := range r.Players {
		if p == userID {
			return true, nil // 이미 참여
		}
	}
	r.Players = append(r.Players, userID)
	if err := r.Save(); err != nil {
		return false, err
	}
	return true, nil
}
