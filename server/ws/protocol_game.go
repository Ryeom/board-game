package ws

import "github.com/Ryeom/board-game/internal/game"

type GameActionRequest struct {
	Action map[string]interface{} `json:"action"`
}

type GameInfoRequest struct {
	GameMode string `json:"gameMode"`
}

type GameSyncResponse struct {
	RoomID    string    `json:"roomId"`
	GameMode  game.Mode `json:"gameMode"`
	GameState any       `json:"gameState"`
}
