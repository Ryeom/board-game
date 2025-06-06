package test

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Ryeom/board-game/server/ws"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func startTestServer(t *testing.T) (*httptest.Server, string) {
	e := echo.New()
	e.GET("/ws", func(c echo.Context) error {
		return ws.Websocket(c)
	})

	ts := httptest.NewServer(e)
	u, err := url.Parse(ts.URL)
	assert.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"

	return ts, u.String()
}

func TestWebSocketConnection(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL+"?id=testUser&name=Tester", nil)
	assert.NoError(t, err)
	defer conn.Close()

	type Event struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}

	event := Event{
		Type: "system.ping",
	}
	b, _ := json.Marshal(event)
	_ = conn.WriteMessage(websocket.TextMessage, b)

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err)

	var res map[string]string
	err = json.Unmarshal(msg, &res)
	assert.NoError(t, err)
	assert.Equal(t, "pong", res["type"])
	assert.Equal(t, "pong", res["message"])
}

func TestRoomCreation(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL+"?id=creator&name=Alice", nil)
	assert.NoError(t, err)
	defer conn.Close()

	type Event struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data,omitempty"`
	}

	event := Event{
		Type: "room.create",
	}
	b, _ := json.Marshal(event)
	_ = conn.WriteMessage(websocket.TextMessage, b)

	for i := 0; i < 2; i++ { // expect room_created and room_list
		_, msg, err := conn.ReadMessage()
		assert.NoError(t, err)
		assert.True(t, strings.Contains(string(msg), "room"))
	}
}
