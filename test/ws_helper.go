// test/ws_helper.go
package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

type WSEvent struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data,omitempty"`
	RoomID    string      `json:"roomId,omitempty"`
	Name      string      `json:"name,omitempty"`
	Message   string      `json:"message,omitempty"`
	Success   bool        `json:"success,omitempty"`
	Code      interface{} `json:"code,omitempty"`
	ErrorCode string      `json:"errorCode,omitempty"`
	Action    string      `json:"action,omitempty"`
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
	assert.NoError(t, err, "Failed to unmarshal message: %s", string(msg)) // 실패 시 메시지 출력
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
	conn, _, err := dialer.Dial(wsURL+"?id="+userID+"&name="+userName, nil) // id, name 쿼리 파라미터는 websocket.go에서 사용하지 않으므로 제거하거나 유의
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
