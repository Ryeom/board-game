package room

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
)

type GameMode string

const (
	GameModeHanabi GameMode = "hanabi"
)

type Room struct {
	ID        string    `json:"id"`
	Host      string    `json:"host"`
	Players   []string  `json:"players"`
	GameMode  GameMode  `json:"gameMode"`
	CreatedAt time.Time `json:"createdAt"`
}

func Initialize() {
}

func CreateRoom(ctx context.Context, roomID string, hostID string) *Room {
	r := &Room{
		ID:        roomID,
		Host:      hostID,
		Players:   []string{hostID},
		GameMode:  GameModeHanabi,
		CreatedAt: time.Now(),
	}
	if err := r.Save(ctx); err != nil {
		return nil
	}
	return r
}

func GetRoom(ctx context.Context, roomID string) (*Room, bool) {
	val, err := redisutil.RoomClient.Get(ctx, "room:"+roomID).Result()
	if err != nil {
		return nil, false
	}
	var room Room
	if err := json.Unmarshal([]byte(val), &room); err != nil {
		return nil, false
	}
	return &room, true
}

func DeleteRoom(ctx context.Context, roomID string) error {
	err := redisutil.RoomClient.Del(ctx, "room:"+roomID).Err()
	if err != nil {
		return fmt.Errorf("failed to delete room %s: %w", roomID, err)
	}
	return nil
}

func ListRooms(ctx context.Context) []*Room {
	keys, err := redisutil.RoomClient.Keys(ctx, "room:*").Result()
	if err != nil {
		return nil
	}
	var rooms []*Room
	for _, key := range keys {
		val, err := redisutil.RoomClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var room Room
		if err := json.Unmarshal([]byte(val), &room); err == nil {
			rooms = append(rooms, &room)
		}
	}
	return rooms
}

func (r *Room) Save(ctx context.Context) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return redisutil.RoomClient.Set(ctx, "room:"+r.ID, b, 0).Err()
}

func (r *Room) Join(ctx context.Context, userID string) (bool, error) {
	for _, p := range r.Players {
		if p == userID {
			return true, nil // 이미 참여
		}
	}
	r.Players = append(r.Players, userID)
	if err := r.Save(ctx); err != nil {
		return false, err
	}
	return true, nil
}
