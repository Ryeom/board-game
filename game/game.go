package game

import (
	"github.com/Ryeom/board-game/game/room"
)

var RoomManager = room.NewRoomManager()

func success(data any) map[string]any {
	return map[string]any{
		"status":  "success",
		"data":    data,
		"message": nil,
	}
}

func failure(message string, code int) (int, map[string]any) {
	return code, map[string]any{
		"status":  "error",
		"data":    nil,
		"message": message,
	}
}
