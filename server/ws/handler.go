package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Ryeom/board-game/internal/util"
	"github.com/Ryeom/board-game/room"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Websocket(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	socketId := generateSocketID(c, conn.RemoteAddr())
	ctx := context.Background()
	user := NewUserSession(socketId, "", "", c.RealIP(), c.Request().UserAgent(), false, conn)

	// 최초 identify 메시지 수신
	var initData websocketInitData
	if _, msg, err := conn.ReadMessage(); err != nil {
		return err
	} else if err := json.Unmarshal(msg, &initData); err != nil {
		return err
	}

	if initData.Type != "identify" {
		return echo.NewHTTPError(http.StatusBadRequest, "expected identify event")
	}
	user.Name = initData.Name

	if cookie, err := c.Cookie("user_name"); err == nil {
		user.Name = cookie.Value
	}

	if err := SaveUserSession(ctx, user); err != nil {
		return err
	}

	fmt.Printf(
		"[Connected] ID: %s | Name: %s | IP: %s | Time: %s\n",
		user.ID, user.Name, user.IP, user.ConnectedAt.Format(time.RFC3339),
	)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(time.Now(), "❌ Disconnected:", err)
			_ = DeleteUserSession(ctx, socketId)
			fmt.Printf(
				"[Disconnected] ID: %s | Name: %s | Room: %s | LastPingAt: %s\n",
				user.ID, user.Name, user.RoomID, user.LastPingAt.Format(time.RFC3339),
			)
			break
		}

		var event SocketEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			fmt.Println(time.Now(), "⚠️ invalid message format:", err)
			continue
		}

		handleEvent(user, event)
	}
	return nil
}

func handleEvent(user *UserSession, event SocketEvent) {
	ctx := context.Background()
	switch event.Type {
	case "create_room":
		rid := "room:" + util.GetUUID()
		host := &userSessionWrapper{user}
		r := room.CreateRoom(ctx, rid, host)
		user.Conn.WriteJSON(map[string]any{
			"type": "room_created",
			"data": map[string]string{
				"roomId": r.ID,
			},
		})
		rooms := room.ListRooms(ctx)
		user.Conn.WriteJSON(map[string]any{
			"type": "room_list",
			"data": rooms,
		})
	case "join_room":
		r, ok := room.JoinRoom(ctx, event.RoomID, user.ID)
		if !ok {
			user.Conn.WriteJSON(map[string]any{
				"type":    "error",
				"message": "room not found",
			})
			return
		}
		user.RoomID = r.ID
		user.Conn.WriteJSON(map[string]any{
			"type": "room_joined",
			"data": r,
		})
		GlobalBroadcaster.BroadcastToRoom(user.RoomID, map[string]any{
			"type": "user_joined",
			"data": map[string]string{
				"userId":   user.ID,
				"userName": user.Name,
			},
		})
	case "start_game":
		fmt.Println("🎮 start game in room:", event.RoomID)
		// TODO: room.GetRoom(event.RoomID).Engine.StartGame()
	case "get_room_list":
		rooms := room.ListRooms(ctx)
		user.Conn.WriteJSON(map[string]any{
			"type": "room_list",
			"data": rooms,
		})
	default:
		fmt.Println("⚠️ unknown event type:", event.Type)
	}
}

type websocketInitData struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type SocketEvent struct {
	Type   string                 `json:"type"`
	RoomID string                 `json:"roomId"`
	Name   string                 `json:"name"`
	Data   map[string]interface{} `json:"data"`
}

type userSessionWrapper struct {
	*UserSession
}

func (u *userSessionWrapper) GetID() string    { return u.ID }
func (u *userSessionWrapper) GetName() string  { return u.Name }
func (u *userSessionWrapper) IsHostUser() bool { return u.IsHost }

func generateSocketID(c echo.Context, addr net.Addr) string {
	ip := c.RealIP()
	remoteIP := addr.String()
	return ip + "_" + remoteIP + "_" + time.Now().Format("20060102150405.000")
}
