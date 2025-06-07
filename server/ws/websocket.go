package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"net"
	"net/http"
	"time"
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
	connectedUser := user.NewUserSession(socketId, "", "", c.RealIP(), c.Request().UserAgent(), false, conn)

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
	connectedUser.Name = initData.Name

	if cookie, err := c.Cookie("user_name"); err == nil {
		connectedUser.Name = cookie.Value
	}

	if err := user.SaveUserSession(ctx, connectedUser); err != nil {
		return err
	}

	fmt.Printf(
		"[Connected] ID: %s | Name: %s | IP: %s | Time: %s\n",
		connectedUser.ID, connectedUser.Name, connectedUser.IP, connectedUser.ConnectedAt.Format(time.RFC3339),
	)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(time.Now(), "❌ Disconnected:", err)
			_ = user.DeleteUserSession(ctx, socketId)
			fmt.Printf(
				"[Disconnected] ID: %s | Name: %s | Room: %s | LastPingAt: %s\n",
				connectedUser.ID, connectedUser.Name, connectedUser.RoomID, connectedUser.LastPingAt.Format(time.RFC3339),
			)
			break
		}

		var event SocketEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			fmt.Println(time.Now(), "⚠️ invalid message format:", err)
			continue
		}

		dispatchSocketEvent(ctx, connectedUser, event)
	}
	return nil
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
	Filter map[string]interface{} `json:"filter"`
}

type userSessionWrapper struct {
	*user.Session
}

func (u *userSessionWrapper) GetID() string    { return u.ID }
func (u *userSessionWrapper) GetName() string  { return u.Name }
func (u *userSessionWrapper) IsHostUser() bool { return u.IsHost }

func generateSocketID(c echo.Context, addr net.Addr) string {
	ip := c.RealIP()
	remoteIP := addr.String()
	return ip + "_" + remoteIP + "_" + time.Now().Format("20060102150405.000")
}
