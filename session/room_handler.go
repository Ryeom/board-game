package session

import (
	"github.com/Ryeom/board-game/game"
	"github.com/Ryeom/board-game/game/room"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

func CreateRoom(c echo.Context) error {
	var req struct {
		RoomID   string `json:"roomId"`
		HostID   string `json:"hostId"`
		HostName string `json:"hostName"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(Failure("invalid request", http.StatusBadRequest))
	}

	host := &UserSession{
		ID:     req.HostID,
		Name:   req.HostName,
		IsHost: true,
	}
	r := game.RoomManager.CreateRoom(req.RoomID, host)

	return c.JSON(Success(map[string]any{
		"roomId":      r.ID,
		"gameMode":    r.GameMode,
		"createdAt":   r.CreatedAt,
		"host":        host.Name,
		"playerCount": len(r.Players),
	}))
}

func GetRoomList(c echo.Context) error {
	rooms := game.RoomManager.ListRooms()

	type roomSummary struct {
		ID        string    `json:"id"`
		Host      string    `json:"host"`
		PlayerNum int       `json:"playerCount"`
		GameMode  string    `json:"gameMode"`
		CreatedAt time.Time `json:"createdAt"`
	}

	summary := make([]roomSummary, 0, len(rooms))
	for _, r := range rooms {
		summary = append(summary, roomSummary{
			ID:        r.ID,
			Host:      r.Host.GetName(),
			PlayerNum: len(r.Players),
			GameMode:  string(r.GameMode),
			CreatedAt: r.CreatedAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data":   summary,
	})
}

func DeleteRoom(c echo.Context) error {
	roomID := c.Param("roomId")
	_, ok := game.RoomManager.GetRoom(roomID)
	if !ok {
		return c.JSON(Failure("room not found", http.StatusNotFound))
	}
	game.RoomManager.DeleteRoom(roomID)
	return c.JSON(Success("room deleted"))
}

func UpdateRoom(c echo.Context) error {
	roomID := c.Param("roomId")

	r, ok := game.RoomManager.GetRoom(roomID)
	if !ok {
		return c.JSON(Failure("room not found", http.StatusNotFound))
	}

	var req struct {
		GameMode room.GameMode `json:"gameMode"`
	}
	if err := c.Bind(&req); err != nil || req.GameMode == "" {
		return c.JSON(Failure("invalid gameMode", http.StatusBadRequest))
	}

	switch req.GameMode {
	case room.GameModeHanabi:
		r.GameMode = req.GameMode
		// TODO: r.Engine = hanabi.NewEngine() ... 등 추후 연결
	default:
		return c.JSON(Failure("unsupported game mode", http.StatusBadRequest))
	}

	return c.JSON(Success("game mode updated"))
}
