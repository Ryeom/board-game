package game

import (
	"github.com/Ryeom/board-game/game/room"
	"github.com/labstack/echo/v4"
	"net/http"
)

var RoomManager *room.RoomManager

func Initialize(e *echo.Echo) {
	RoomManager = room.NewRoomManager()

	registerAPI(e)
}
func registerAPI(e *echo.Echo) {
	e.GET("/api/rooms", listRooms)
	e.GET("/api/rooms/:roomId/players", listPlayers)
	e.POST("/api/rooms", createRoom)
	e.DELETE("/api/rooms/:roomId", deleteRoom)
	e.POST("/api/rooms/:roomId/players/:playerId/ready", markPlayerReady)
	e.DELETE("/api/rooms/:roomId/players/:playerId", removePlayer)
}

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
	e.GET("/api/rooms", listRooms)
	e.GET("/api/rooms/:roomId/players", listPlayers)
	e.POST("/api/rooms", createRoom)
	e.DELETE("/api/rooms/:roomId", deleteRoom)
	e.POST("/api/rooms/:roomId/players/:playerId/ready", markPlayerReady)
	e.DELETE("/api/rooms/:roomId/players/:playerId", removePlayer)
	e.POST("/api/rooms/:roomId/mode", updateGameMode)
}

func updateGameMode(c echo.Context) error {
	roomId := c.Param("roomId")
	r, ok := RoomManager.GetRoom(roomId)
	if !ok {
		return c.JSON(failure("room not found", http.StatusNotFound))
	}
	var req struct {
		GameMode room.GameMode `json:"gameMode"`
	}
	if err := c.Bind(&req); err != nil || req.GameMode == "" {
		return c.JSON(failure("invalid gameMode", http.StatusBadRequest))
	}

	switch req.GameMode {
	case room.GameModeHanabi:
		r.GameMode = req.GameMode
		// 추후 수정
		//host := room.NewAttender(room., "", true)
		return c.JSON(http.StatusOK, success("game mode updated to hanabi"))
	default:
		return c.JSON(failure("unsupported game mode", http.StatusBadRequest))
	}
}

func listRooms(c echo.Context) error {
	rooms := RoomManager.ListRooms()
	resp := []map[string]any{}
	for _, r := range rooms {
		resp = append(resp, map[string]any{
			"id":          r.ID,
			"gameMode":    r.GameMode,
			"playerCount": len(r.Players),
			"createdAt":   r.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, success(resp))
}

func listPlayers(c echo.Context) error {
	roomId := c.Param("roomId")
	r, ok := RoomManager.GetRoom(roomId)
	if !ok {
		return c.JSON(failure("room not found", http.StatusNotFound))
	}
	players := []map[string]any{}
	for _, p := range r.Players {
		players = append(players, map[string]any{
			"id":     p.ID,
			"name":   p.Name,
			"isHost": p.IsHost,
			"ready":  p.Ready,
		})
	}
	return c.JSON(http.StatusOK, success(players))
}

func createRoom(c echo.Context) error {
	var req struct {
		RoomID   string `json:"roomId"`
		HostName string `json:"hostName"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(failure("invalid request", http.StatusBadRequest))
	}
	if req.RoomID == "" || req.HostName == "" {
		return c.JSON(failure("roomId and hostName are required", http.StatusBadRequest))
	}
	if r, ok := RoomManager.GetRoom(req.RoomID); ok {
		for _, p := range r.Players {
			if p.Name == req.HostName {
				return c.JSON(failure("name already taken in this room", http.StatusConflict))
			}
		}
	}

	host := room.NewAttender(req.RoomID+"-host", req.HostName, true)
	r := RoomManager.CreateRoom(req.RoomID, host, room.GameModeHanabi, nil)
	// 여기서 랜덤 게임모드 설정

	return c.JSON(http.StatusOK, success(map[string]any{
		"roomId":      r.ID,
		"gameMode":    r.GameMode,
		"createdAt":   r.CreatedAt,
		"host":        host.Name,
		"playerCount": len(r.Players),
	}))
}

func deleteRoom(c echo.Context) error {
	roomId := c.Param("roomId")
	RoomManager.DeleteRoom(roomId)
	return c.JSON(http.StatusOK, success("room deleted"))
}

func markPlayerReady(c echo.Context) error {
	roomId := c.Param("roomId")
	playerId := c.Param("playerId")
	r, ok := RoomManager.GetRoom(roomId)
	if !ok {
		return c.JSON(failure("room not found", http.StatusNotFound))
	}
	for _, p := range r.Players {
		if p.ID == playerId {
			p.Ready = true
			break
		}
	}
	ready := true
	for _, p := range r.Players {
		if !p.Ready {
			ready = false
			break
		}
	}
	if ready && r.Engine != nil {
		r.Engine.StartGame()
	}
	return c.JSON(http.StatusOK, success("player ready"))
}

func removePlayer(c echo.Context) error {
	roomId := c.Param("roomId")
	playerId := c.Param("playerId")
	r, ok := RoomManager.GetRoom(roomId)
	if !ok {
		return c.JSON(failure("room not found", http.StatusNotFound))
	}
	for i, p := range r.Players {
		if p.ID == playerId {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}
	return c.JSON(http.StatusOK, success("player removed"))
}
