package room

import (
	"context"
	"time"
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

var controlManager = NewManager()

func CreateRoom(ctx context.Context, id string, host Player) *Room {
	return controlManager.CreateRoom(ctx, id, host)
}

func GetRoom(ctx context.Context, id string) (*Room, bool) {
	return controlManager.GetRoom(ctx, id)
}

func DeleteRoom(ctx context.Context, id string) {
	controlManager.DeleteRoom(ctx, id)
}

func ListRooms(ctx context.Context) []*Room {
	return controlManager.ListRooms(ctx)
}

func SaveRoom(ctx context.Context, r *Room) error {
	return controlManager.SaveRoom(ctx, r)
}

func JoinRoom(ctx context.Context, roomID string, userID string) (*Room, bool) {
	return controlManager.JoinRoom(ctx, roomID, userID)
}
