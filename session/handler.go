// session/handler.go
package session

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func RegisterWebSocket(e *echo.Echo) {
	e.GET("/ws", func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()

		socketID := c.RealIP() + conn.RemoteAddr().String()

		user := &UserSession{
			ID:         socketID,
			Connection: conn,
		}
		Register(user)
		fmt.Println("🔌 Connected:", socketID)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("❌ Disconnected:", err)
				Unregister(socketID)
				break
			}

			var event struct {
				Type   string         `json:"type"`
				RoomID string         `json:"roomId"`
				Name   string         `json:"name"`
				Data   map[string]any `json:"data"`
			}
			if err := json.Unmarshal(msg, &event); err != nil {
				fmt.Println("⚠️ invalid message format:", err)
				continue
			}

			handleEvent(user, event)
		}
		return nil
	})
}

func handleEvent(user *UserSession, event struct {
	Type   string
	RoomID string
	Name   string
	Data   map[string]any
}) {
	switch event.Type {
	case "create_room":
		fmt.Println("🛠 creating room:", event.RoomID, "by", event.Name)
		// TODO: room.CreateRoom(event.RoomID, user)
	case "join_room":
		fmt.Println("👥 joining room:", event.RoomID, "by", event.Name)
		// TODO: room.JoinRoom(event.RoomID, user)
	case "start_game":
		fmt.Println("🎮 start game in room:", event.RoomID)
		// TODO: room.GetRoom(event.RoomID).Engine.StartGame()
	default:
		fmt.Println("⚠️ unknown event type:", event.Type)
	}
}
