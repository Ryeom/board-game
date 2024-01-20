package server

import (
	"encoding/json"
	"fmt"
	"github.com/Ryeom/hanabi/hanabi"
	"github.com/Ryeom/hanabi/log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func SocketHandler(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Logger.Error("Error Upgrade : ", err)
		return err
	}
	log.Logger.Info("Connected socket", c.RealIP())
	fmt.Println("Connected socket", c.RealIP())
	defer ws.Close()
	// TODO : 참여자 생성하지않았으면 참여자 생성 혹은 ip가 중복이 있다면 ip할당된 참여자목록 찾아서 해당 참여자로 재연결
	fmt.Println("Connected socket", c.RealIP())

	for {
		var err error
		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error(err)
		}

		var act hanabi.Action
		err = json.Unmarshal(msg, &act)
		if err != nil {
			c.Logger().Error(err)
		}
		act.Response = make(chan string)
		hanabi.ActionExecution <- &act

		resp := <-act.Response
		// write
		err = ws.WriteMessage(websocket.TextMessage, []byte(resp))
		if err != nil {
			c.Logger().Error(err)
		}
	}
}
