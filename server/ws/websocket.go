package ws

import (
	"encoding/json"
	"fmt"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"net"
	"net/http"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

var activeSessions sync.Map

func ActiveSessions() *sync.Map {
	return &activeSessions
}

func Websocket(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	tempSocketID := generateSocketID(c, conn.RemoteAddr())

	currentUserSession := user.NewUserSession(tempSocketID, "", "", c.RealIP(), c.Request().UserAgent(), false, conn)

	activeSessions.Store(currentUserSession.ID, currentUserSession)

	fmt.Printf(
		"[Initial Conn] ID: %s | IP: %s | Time: %s\n",
		currentUserSession.ID, currentUserSession.IP, currentUserSession.ConnectedAt.Format(time.RFC3339),
	)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if val, ok := activeSessions.Load(currentUserSession.ID); ok {
				currentUserSession = val.(*user.Session)
			}
			log.Logger.Infof("WebSocket Disconnected for ID: %s, Name: %s, Error: %v", currentUserSession.ID, currentUserSession.Name, err)
			HandleUserDisconnect(c.Request().Context(), currentUserSession, SocketEvent{Type: "user.disconnect"})
			break
		}

		var event SocketEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			log.Logger.Warningf("WebSocket invalid message format from ID: %s, Error: %v, Message: %s", currentUserSession.ID, err, string(msg))
			sendError(currentUserSession, resp.ErrorCodeWSInvalidMessageFormat)
			continue
		}

		currentUserSession.LastPingAt = time.Now()
		activeSessions.Store(currentUserSession.ID, currentUserSession)
		dispatchSocketEvent(c.Request().Context(), currentUserSession, event)
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

func generateSocketID(c echo.Context, addr net.Addr) string {
	ip := c.RealIP()
	remoteIP := addr.String()
	return ip + "_" + remoteIP + "_" + time.Now().Format("20060102150405.000")
}
