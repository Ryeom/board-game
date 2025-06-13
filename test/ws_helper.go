// test/ws_helper.go (새 파일)
package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

type WSEvent struct {
	Type   string      `json:"type"`
	Data   interface{} `json:"data,omitempty"`
	RoomID string      `json:"roomId,omitempty"`
	Name   string      `json:"name,omitempty"`
}

func SendEvent(t *testing.T, conn *websocket.Conn, event WSEvent) {
	b, err := json.Marshal(event)
	assert.NoError(t, err, "Failed to marshal event")
	err = conn.WriteMessage(websocket.TextMessage, b)
	assert.NoError(t, err, "Failed to write message")
}

func ReadEvent(t *testing.T, conn *websocket.Conn, timeout time.Duration) WSEvent {
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err, "Failed to read message")

	var event WSEvent
	err = json.Unmarshal(msg, &event)
	assert.NoError(t, err, "Failed to unmarshal message")
	return event
}

func IdentifyUser(t *testing.T, conn *websocket.Conn, userID, userName string) {
	event := WSEvent{
		Type: "identify",
		Data: map[string]interface{}{
			"userId":   userID,
			"userName": userName,
		},
	}
	SendEvent(t, conn, event)
}

func ConnectAndIdentify(t *testing.T, wsURL, userID, userName string) *websocket.Conn {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL+"?id="+userID+"&name="+userName, nil)
	assert.NoError(t, err)

	IdentifyUser(t, conn, userID, userName)

	return conn
}

func GetNthMessage(t *testing.T, conn *websocket.Conn, n int, timeout time.Duration) []WSEvent {
	messages := make([]WSEvent, 0)
	for i := 0; i < n; i++ {
		messages = append(messages, ReadEvent(t, conn, timeout))
	}
	return messages
}
