package room

import (
	"encoding/json"
	"fmt"
	"github.com/Ryeom/board-game/game/hanabi"
	"net/http"

	"github.com/Ryeom/board-game/game/room/types"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func RegisterWebSocketHandler(e *echo.Echo, manager *RoomManager) {
	e.GET("/ws/:roomId", func(c echo.Context) error {
		roomId := c.Param("roomId")

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()

		socketID := c.RealIP() + conn.RemoteAddr().String()

		_, initMsg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var initData struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(initMsg, &initData); err != nil {
			return err
		}

		r, ok := manager.GetRoom(roomId)
		if !ok {
			// 없으면 새로 생성
			host := NewAttender(socketID, initData.Name, true)
			host.Connection = conn
			hanabiEngine := hanabi.NewEngine(
				[]string{host.ID},
				func(ids []string, state any) {
					for _, p := range r.Players {
						for _, id := range ids {
							if p.ID == id {
								SendWSJSON(p, WSResponse{Status: "success", Data: state, Message: nil})
							}
						}
					}
				},
				func(state any) { r.State = state },
				func() []string {
					ids := make([]string, len(r.Players))
					for i, p := range r.Players {
						ids[i] = p.ID
					}
					return ids
				},
			)
			r = manager.CreateRoom(roomId, host, GameModeHanabi, hanabiEngine)
		} else {
			player := NewAttender(socketID, initData.Name, false)
			player.Connection = conn
			r.Players = append(r.Players, player)
		}

		fmt.Printf("🎉 %s joined room %s\n", initData.Name, roomId)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("연결 종료:", err)
				break
			}

			var event types.Event
			if err := json.Unmarshal(msg, &event); err != nil {
				fmt.Println("❌ 잘못된 이벤트:", err)
				continue
			}

			if r.Engine == nil {
				fmt.Println("🚫 아직 게임이 시작되지 않음")
				continue
			}

			if event.Type == types.EventStartGame {
				r.Engine.StartGame()
				continue
			}

			err = r.Engine.HandleEvent(event)
			if err != nil {
				fmt.Println("🚨 이벤트 처리 중 오류:", err)
			}
		}

		return nil
	})
}
