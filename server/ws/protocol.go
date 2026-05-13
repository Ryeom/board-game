package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	resp "github.com/Ryeom/board-game/internal/response"
)

type EventType string

type SocketEvent struct {
	Type   EventType              `json:"type"`
	Data   map[string]interface{} `json:"data,omitempty"`
	Filter map[string]interface{} `json:"filter,omitempty"`
}

type WebSocketResult struct {
	Type       EventType   `json:"type"`
	Data       interface{} `json:"data,omitempty"`
	Message    string      `json:"message"`
	Success    bool        `json:"success"`
	StatusCode int         `json:"code,omitempty"`
	ErrorCode  string      `json:"errorCode,omitempty"`
	Action     string      `json:"action,omitempty"`
	Timestamp  time.Time   `json:"timestamp,omitempty"`
}

func bindEventData[T any](event SocketEvent, dest *T) error {
	if event.Data == nil {
		return fmt.Errorf("missing data")
	}
	b, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func createWebSocketResult(eventType EventType, data interface{}, resultMsgCode, lang string) *WebSocketResult {
	msgData, found := resp.GetDefineCode(resultMsgCode, lang)
	if !found {
		msgData.Message = fmt.Sprintf("Unknown response code: %s", resultMsgCode)
		msgData.HttpStatus = http.StatusOK
		msgData.Action = "Please contact support."
	}

	return &WebSocketResult{
		Type:       eventType,
		Data:       data,
		Message:    msgData.Message,
		Success:    eventType != EventError,
		StatusCode: msgData.HttpStatus,
		ErrorCode:  resultMsgCode,
		Action:     msgData.Action,
		Timestamp:  time.Now(),
	}
}
