package http

import (
	"github.com/Ryeom/board-game/user"
	"net/http"
	"time"

	"github.com/Ryeom/board-game/room"
	"github.com/labstack/echo/v4"
)

func CreateRoom(c echo.Context) error {
	var req struct {
		RoomID   string `json:"roomId"`
		HostID   string `json:"hostId"`
		HostName string `json:"hostName"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"status": "fail", "message": "invalid request"})
	}

	host := &user.Session{
		ID:     req.HostID,
		Name:   req.HostName,
		IsHost: true,
	}
	r := room.CreateRoom(c.Request().Context(), req.RoomID, req.HostID)

	return c.JSON(http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"roomId":      r.ID,
			"gameMode":    r.GameMode,
			"createdAt":   r.CreatedAt,
			"host":        host.Name,
			"playerCount": len(r.Players),
		},
	})
}

func GetRoomList(c echo.Context) error {
	type roomSummary struct {
		ID        string    `json:"id"`
		Host      string    `json:"host"`
		PlayerNum int       `json:"playerCount"`
		GameMode  string    `json:"gameMode"`
		CreatedAt time.Time `json:"createdAt"`
	}

	rooms := room.ListRooms(c.Request().Context())
	summary := make([]roomSummary, 0, len(rooms))
	for _, r := range rooms {
		summary = append(summary, roomSummary{
			ID:        r.ID,
			Host:      r.Host,
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
	_, ok := room.GetRoom(c.Request().Context(), roomID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]any{"status": "fail", "message": "room not found"})
	}
	room.DeleteRoom(c.Request().Context(), roomID)
	return c.JSON(http.StatusOK, map[string]any{"status": "success", "message": "room deleted"})
}

func UpdateRoom(c echo.Context) error {
	roomID := c.Param("roomId")

	r, ok := room.GetRoom(c.Request().Context(), roomID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]any{"status": "fail", "message": "room not found"})
	}
	r.Save()
	var req struct {
		GameMode room.GameMode `json:"gameMode"`
	}
	if err := c.Bind(&req); err != nil || req.GameMode == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"status": "fail", "message": "invalid gameMode"})
	}

	switch req.GameMode {
	case room.GameModeHanabi:
		//r.GameMode = req.GameMode
		//_ = room.SaveRoom(c.Request().Context(), r)
		// TODO: r.Engine = hanabi.NewEngine() ... 등 추후 연결
	default:
		return c.JSON(http.StatusBadRequest, map[string]any{"status": "fail", "message": "unsupported game mode"})
	}

	return c.JSON(http.StatusOK, map[string]any{"status": "success", "message": "game mode updated"})
}
