package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
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
	//ctx := context.Background() // WebSocket 핸들러 컨텍스트

	// 최초에는 기본 세션 정보만 생성. RoomID 등은 아직 모름.
	connectedUser := user.NewUserSession(socketId, "", "", c.RealIP(), c.Request().UserAgent(), false, conn)

	// 최초 identify 메시지 수신
	var initData websocketInitData
	if _, msg, err := conn.ReadMessage(); err != nil {
		fmt.Println(time.Now(), "❌ WebSocket Init Read Error:", err)
		return err // 초기 메시지 수신 실패 시 연결 종료
	} else if err := json.Unmarshal(msg, &initData); err != nil {
		fmt.Println(time.Now(), "❌ WebSocket Init Unmarshal Error:", err)
		return errors.New(resp.ErrorCodeAuthInvalidRequest)
	}

	if initData.Type != "identify" {
		fmt.Println(time.Now(), "❌ WebSocket Init - Expected 'identify' event, got:", initData.Type)
		return errors.New(resp.ErrorCodeWSExpectedIdentify)
	}

	connectedUser.Name = initData.Name

	if cookie, err := c.Cookie("user_name"); err == nil {
		connectedUser.Name = cookie.Value
	}

	if err := user.SaveUserSession(connectedUser); err != nil {
		log.Logger.Errorf("Websocket - Initial SaveUserSession error: %v", err)
		return errors.New(resp.ErrorCodeWSInitialSessionSaveFailed)
	}

	fmt.Printf(
		"[Connected] ID: %s | Name: %s | IP: %s | Time: %s\n",
		connectedUser.ID, connectedUser.Name, connectedUser.IP, connectedUser.ConnectedAt.Format(time.RFC3339),
	)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Logger.Infof("WebSocket Disconnected for ID: %s, Name: %s, Error: %v", connectedUser.ID, connectedUser.Name, err)
			HandleUserDisconnect(c.Request().Context(), connectedUser, SocketEvent{Type: "user.disconnect"})
			break
		}

		var event SocketEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			log.Logger.Warningf("WebSocket invalid message format from ID: %s, Error: %v, Message: %s", connectedUser.ID, err, string(msg))
			sendError(connectedUser, resp.ErrorCodeWSInvalidMessageFormat)
			continue
		}

		dispatchSocketEvent(c.Request().Context(), connectedUser, event)
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
