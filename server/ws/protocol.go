package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Ryeom/board-game/internal/game"
	resp "github.com/Ryeom/board-game/internal/response"
)

type EventType string

const (
	EventRoomCreate EventType = "room.create"
	EventRoomJoin   EventType = "room.join"
	EventRoomLeave  EventType = "room.leave"
	EventRoomList   EventType = "room.list"
	EventRoomUpdate EventType = "room.update"
	EventRoomReady  EventType = "room.ready"
	EventRoomKick   EventType = "room.kick"

	EventUserIdentify     EventType = "user.identify"
	EventUserUpdate       EventType = "user.update"
	EventUserDisconnect   EventType = "user.disconnect"
	EventUserDisconnected EventType = "user.disconnected"
	EventUserStatus       EventType = "user.status"
	EventUserLeft         EventType = "user.left"
	EventUserKicked       EventType = "user.kicked"
	EventUserReconnected  EventType = "user.reconnected"

	EventGameStart           EventType = "game.start"
	EventGameStarted         EventType = "game.started"
	EventGameEnd             EventType = "game.end"
	EventGameEnded           EventType = "game.ended"
	EventGameAction          EventType = "game.action"
	EventGameActionSync      EventType = "game.action.sync"
	EventGameActionSucceeded EventType = "game.action.succeeded"
	EventGameSync            EventType = "game.sync"
	EventGamePause           EventType = "game.pause"
	EventGameInfo            EventType = "game.info"
	EventGameTimerStarted    EventType = "game.timer.started"
	EventGameTimerReset      EventType = "game.timer.reset"
	EventGameTimerExpired    EventType = "game.timer.expired"

	EventChatSend    EventType = "chat.send"
	EventChatMessage EventType = "chat.message"
	EventChatHistory EventType = "chat.history"
	EventChatMute    EventType = "chat.mute"

	EventSystemPing   EventType = "system.ping"
	EventSystemPong   EventType = "system.pong"
	EventSystemError  EventType = "system.error"
	EventSystemNotice EventType = "system.notice"
	EventSystemSync   EventType = "system.sync"
	EventError        EventType = "error"
)

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

type RoomCreateRequest struct {
	RoomName   string `json:"roomName"`
	Password   string `json:"password,omitempty"`
	MaxPlayers int    `json:"maxPlayers"`
}

type RoomJoinRequest struct {
	RoomID   string `json:"roomId"`
	Password string `json:"password,omitempty"`
}

type RoomKickRequest struct {
	UserID string `json:"userId"`
}

type RoomCreateResponse struct {
	RoomID     string        `json:"roomId"`
	RoomName   string        `json:"roomName"`
	MaxPlayers int           `json:"maxPlayers"`
	RoomList   []RoomSummary `json:"roomList"`
}

type RoomLeaveResponse struct {
	RoomID  string `json:"roomId"`
	NewHost string `json:"newHost"`
	Deleted bool   `json:"deleted"`
}

type RoomListResponse struct {
	Rooms []RoomSummary `json:"rooms"`
}

type RoomSummary struct {
	ID          string    `json:"id"`
	RoomName    string    `json:"roomName"`
	Host        string    `json:"host"`
	PlayerCount int       `json:"playerCount"`
	MaxPlayers  int       `json:"maxPlayers"`
	GameMode    string    `json:"gameMode"`
	HasPassword bool      `json:"hasPassword"`
	CreatedAt   time.Time `json:"createdAt"`
}

type UserIdentifyRequest struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
}

type UserUpdateRequest struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

type UserStatusRequest struct {
	UserID string `json:"userId"`
}

type GameActionRequest struct {
	Action map[string]interface{} `json:"action"`
}

type GameInfoRequest struct {
	GameMode string `json:"gameMode"`
}

type GameSyncResponse struct {
	RoomID    string    `json:"roomId"`
	GameMode  game.Mode `json:"gameMode"`
	GameState any       `json:"gameState"`
}

type ChatSendRequest struct {
	Message string `json:"message"`
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
