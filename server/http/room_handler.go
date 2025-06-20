package http

import (
	"github.com/Ryeom/board-game/internal/domain/room"
	apperr "github.com/Ryeom/board-game/internal/errors"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
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
		log.Logger.Errorf("CreateRoom - Bind Error: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeRoomInvalidRequest, err)
	}

	host := &user.Session{
		ID:     req.HostID,
		Name:   req.HostName,
		IsHost: true,
	}
	r := room.CreateRoom(c.Request().Context(), req.RoomID, req.HostID)

	return c.JSON(http.StatusOK, Success(map[string]any{
		"roomId":      r.ID,
		"gameMode":    r.GameMode,
		"createdAt":   r.CreatedAt,
		"host":        host.Name,
		"playerCount": len(r.Players),
	}, "게임방이 성공적으로 생성되었습니다."))
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

	return c.JSON(http.StatusOK, Success(summary, "게임방 목록 조회 성공"))
}

// DeleteRoom 함수는 특정 게임방을 삭제합니다.
func DeleteRoom(c echo.Context) error {
	roomID := c.Param("roomId")
	_, ok := room.GetRoom(c.Request().Context(), roomID)
	if !ok {
		return apperr.NotFound(apperr.ErrorCodeRoomNotFound, nil)
	}
	room.DeleteRoom(c.Request().Context(), roomID)
	return c.JSON(http.StatusOK, Success(nil, "게임방이 성공적으로 삭제되었습니다."))
}

func UpdateRoom(c echo.Context) error {
	roomID := c.Param("roomId")

	r, ok := room.GetRoom(c.Request().Context(), roomID)
	if !ok {
		return apperr.NotFound(apperr.ErrorCodeRoomNotFound, nil)
	}

	err := r.Save()
	if err != nil {
		log.Logger.Errorf("UpdateRoom - Save: %v", err)
		return apperr.InternalServerError(apperr.ErrorCodeDefaultInternalServerError, nil)
	}

	var req struct {
		GameMode room.GameMode `json:"gameMode"`
	}
	if err := c.Bind(&req); err != nil || req.GameMode == "" {
		log.Logger.Errorf("UpdateRoom - Bind Error or Empty GameMode: %v", err)
		return apperr.BadRequest(apperr.ErrorCodeRoomInvalidRequest, err)
	}

	switch req.GameMode {
	case room.GameModeHanabi:
		// r.GameMode = req.GameMode
		// _ = room.SaveRoom(c.Request().Context(), r)
		// TODO: r.Engine = hanabi.NewEngine() ... 등 추후 연결
	default:
		// 지원하지 않는 게임 모드인 경우
		return apperr.BadRequest(apperr.ErrorCodeRoomUnsupportedGameMode, nil)
	}

	return c.JSON(http.StatusOK, Success(nil, "게임 모드가 성공적으로 업데이트되었습니다."))
}
