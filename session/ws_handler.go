package session

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Ryeom/board-game/game/room"

	"github.com/Ryeom/board-game/util"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func registerWebSocket(e *echo.Group) {
	e.GET("/ws", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()

		socketID := c.RealIP() + conn.RemoteAddr().String()

		user := &UserSession{
			ID:         socketID,
			Name:       c.Request().Header.Get("X-User-Name"), // 또는 쿠키에서 꺼내도 됨
			Connection: conn,
		}
		fmt.Println(user)
		cookie, err := c.Cookie("user_name")
		if err == nil {
			user.Name = cookie.Value
		}
		Register(user)
		fmt.Println("🔌 Connected:", user.Name)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println(time.Now(), "❌ Disconnected:", err)
				Unregister(socketID)
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
	})
}

func handleEvent(user *UserSession, event SocketEvent) {
	ctx := context.Background()
	switch event.Type {
	case "create_room":
		rid := "room-" + util.GetUUID()
		host := &userSessionWrapper{user}
		r := room.CreateRoom(ctx, rid, host)
		_ = room.SaveRoom(ctx, r)
		user.Connection.WriteJSON(map[string]any{
			"type": "room_created",
			"data": map[string]string{
				"roomId": r.ID,
			},
		})
		rooms := room.ListRooms(ctx)
		user.Connection.WriteJSON(map[string]any{
			"type": "room_list",
			"data": rooms,
		})
	case "join_room":
		fmt.Println("👥 joining room:", event.RoomID, "by", event.Name)
		if r, ok := room.JoinRoom(ctx, event.RoomID, user.ID); ok {
			user.Connection.WriteJSON(map[string]any{
				"type": "room_joined",
				"data": r,
			})
		} else {
			user.Connection.WriteJSON(map[string]any{
				"type":    "error",
				"message": "room not found",
			})
		}

	case "start_game":
		fmt.Println("🎮 start game in room:", event.RoomID)
		// TODO: room.GetRoom(event.RoomID).Engine.StartGame()

	case "get_room_list":
		rooms := room.ListRooms(ctx)
		user.Connection.WriteJSON(map[string]any{
			"type": "room_list",
			"data": rooms,
		})

	default:
		fmt.Println("⚠️ unknown event type:", event.Type)
	}
}
