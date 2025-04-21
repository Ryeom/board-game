package game

import (
	"github.com/Ryeom/board-game/game/room"
	"github.com/Ryeom/board-game/session"
	"github.com/labstack/echo/v4"
	"net/http"
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

func RegisterAPI(e *echo.Echo) {
	e.POST("/api/rooms", createRoom)
}

func createRoom(c echo.Context) error {
	var req struct {
		RoomID   string `json:"roomId"`
		HostID   string `json:"hostId"`
		HostName string `json:"hostName"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(failure("invalid request", http.StatusBadRequest))
	}

	host := &session.UserSession{
		ID:     req.HostID,
		Name:   req.HostName,
		IsHost: true,
	}
	r := RoomManager.CreateRoom(req.RoomID, host)

	return c.JSON(http.StatusOK, success(map[string]any{
		"roomId":      r.ID,
		"gameMode":    r.GameMode,
		"createdAt":   r.CreatedAt,
		"host":        host.Name,
		"playerCount": len(r.Players),
	}))
}
