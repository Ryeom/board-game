package ws

import (
	"context"
	"fmt"
	"github.com/Ryeom/board-game/internal/game"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"net/http"
	"time"
)

func dispatchSocketEvent(ctx context.Context, user *user.Session, event SocketEvent) {
	handler := getHandler(event.Type)
	handler(ctx, user, event)
}

func getHandler(eventType string) ExecutionEvent {
	if handler, ok := eventHandlers[eventType]; ok {
		return handler
	}
	return HandleDefault
}

type ExecutionEvent func(ctx context.Context, user *user.Session, event SocketEvent)

func mergeHandlers(maps ...map[string]ExecutionEvent) map[string]ExecutionEvent {
	merged := make(map[string]ExecutionEvent)
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

var eventHandlers = mergeHandlers(
	roomEvents,
	userEvents,
	gameEvents,
	chatEvents,
	systemEvents,
)

// 방 관련 이벤트 핸들러
var roomEvents = map[string]ExecutionEvent{
	"room.create": HandleRoomCreate, // 방 생성
	"room.join":   HandleRoomJoin,   // 방 참가
	"room.leave":  HandleRoomLeave,  // 방 나가기
	"room.list":   HandleRoomList,   // 방 목록 조회
	"room.update": HandleRoomUpdate, // 방 설정 변경
	"room.ready":  HandleRoomReady,  // 준비 상태 토글
	"room.kick":   HandleRoomKick,   // 강제 퇴장
	//"room.delete": HandleRoomDelete, // 방 삭제
}

// 유저 관련 이벤트 핸들러
var userEvents = map[string]ExecutionEvent{
	"user.identify":   HandleUserIdentify,   // 유저 초기 식별
	"user.update":     HandleUserUpdate,     // 유저 정보 업데이트
	"user.disconnect": HandleUserDisconnect, // 유저 연결 종료
	"user.status":     HandleUserStatus,     // 유저 상태 조회
}

// 게임 관련 이벤트 핸들러
var gameEvents = map[string]ExecutionEvent{
	"game.start":  HandleGameStart,  // 게임 시작
	"game.end":    HandleGameEnd,    // 게임 종료
	"game.action": HandleGameAction, // 플레이어 행동
	"game.sync":   HandleGameSync,   // 게임 상태 동기화
	"game.pause":  HandleGamePause,  // 게임 일시정지
	"game.info":   HandleGameInfo,   // 게임 설명 출력
}

// 채팅 관련 이벤트 핸들러
var chatEvents = map[string]ExecutionEvent{
	"chat.send":    HandleChatSend,    // 채팅 메시지 전송
	"chat.history": HandleChatHistory, // 채팅 내역 조회
	"chat.mute":    HandleChatMute,    // 유저 채팅 제한
}

// 시스템 관련 이벤트 핸들러
var systemEvents = map[string]ExecutionEvent{
	"system.ping":   HandleSystemPing,   // 핑 체크
	"system.error":  HandleSystemError,  // 에러 전달
	"system.notice": HandleSystemNotice, // 시스템 공지
	"system.sync":   HandleSystemSync,   // 시스템 전체 상태 동기화
}

func sendResult(u *user.Session, eventType string, data interface{}, resultMsgCode string) {
	if u.Conn == nil {
		return
	}
	res := createWebSocketResult(eventType, data, resultMsgCode, "ko")
	_ = u.Conn.WriteJSON(res)
}
func sendError(u *user.Session, resultMsgCode string) {
	if u.Conn == nil {
		return
	}
	res := createWebSocketResult("error", nil, resultMsgCode, "ko")
	_ = u.Conn.WriteJSON(res)
}

func createWebSocketResult(eventType string, data interface{}, resultMsgCode, lang string) *WebSocketResult {
	msgData, found := resp.GetDefineCode(resultMsgCode, lang)
	if !found {
		msgData.Message = fmt.Sprintf("Unknown success code: %s", resultMsgCode)
		msgData.HttpStatus = http.StatusOK
		msgData.Action = "Please contact support."
	}
	success := true
	if eventType == "error" {
		success = false
	}

	res := WebSocketResult{
		Type:      eventType,
		Data:      data,
		Message:   msgData.Message,
		Success:   success,
		Code:      msgData.HttpStatus,
		ErrorCode: resultMsgCode,
		Action:    msgData.Action,
		Timestamp: time.Now(),
	}
	return &res
}

type WebSocketResult struct {
	Type      string      `json:"type"`                // 메시지 타입 (예: "room_created", "error")
	Data      interface{} `json:"data"`                // 실제 전송 데이터
	Message   string      `json:"message"`             // 선택적인 설명 메시지
	Success   bool        `json:"success"`             // 성공여부
	Code      int         `json:"code,omitempty"`      // HTTP Status Code와 유사 (에러 시)
	ErrorCode string      `json:"errorCode,omitempty"` // 내부 에러 코드 (에러 시)
	Action    string      `json:"action,omitempty"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
}

// GameStatePayload WebSocketResult.Data
type GameStatePayload struct {
	RoomId              string      `json:"roomId"`              // 현재 게임이 진행 중인 방의 ID
	GameMode            game.Mode   `json:"gameMode"`            // 현재 게임의 모드 (예: "hanabi")
	GameStatus          game.Status `json:"gameStatus"`          // 현재 게임의 상태
	GameState           game.State  `json:"gameState"`           // 게임 엔진의 실제 상태 객체 (플레이어별 뷰가 적용된 상태)
	CurrentTurnPlayerId string      `json:"currentTurnPlayerId"` // 현재 턴을 진행 중인 플레이어의 ID
	Timestamp           time.Time   `json:"timestamp"`           // 이 데이터가 서버에서 생성된 시각
}
