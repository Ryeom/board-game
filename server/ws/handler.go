package ws

import (
	"context"
	"time"

	"github.com/Ryeom/board-game/internal/game"
	"github.com/Ryeom/board-game/internal/user"
)

func dispatchSocketEvent(ctx context.Context, user *user.Session, event SocketEvent) {
	handler := getHandler(event.Type)
	handler(ctx, user, event)
}

func getHandler(eventType EventType) ExecutionEvent {
	if handler, ok := eventHandlers[eventType]; ok {
		return handler
	}
	return HandleDefault
}

type ExecutionEvent func(ctx context.Context, user *user.Session, event SocketEvent)

func mergeHandlers(maps ...map[EventType]ExecutionEvent) map[EventType]ExecutionEvent {
	merged := make(map[EventType]ExecutionEvent)
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
var roomEvents = map[EventType]ExecutionEvent{
	EventRoomCreate: HandleRoomCreate, // 방 생성
	EventRoomJoin:   HandleRoomJoin,   // 방 참가
	EventRoomLeave:  HandleRoomLeave,  // 방 나가기
	EventRoomList:   HandleRoomList,   // 방 목록 조회
	EventRoomUpdate: HandleRoomUpdate, // 방 설정 변경
	EventRoomReady:  HandleRoomReady,  // 준비 상태 토글
	EventRoomKick:   HandleRoomKick,   // 강제 퇴장
	//"room.delete": HandleRoomDelete, // 방 삭제
}

// 유저 관련 이벤트 핸들러
var userEvents = map[EventType]ExecutionEvent{
	EventUserIdentify:   HandleUserIdentify,   // 유저 초기 식별
	EventUserUpdate:     HandleUserUpdate,     // 유저 정보 업데이트
	EventUserDisconnect: HandleUserDisconnect, // 유저 연결 종료
	EventUserStatus:     HandleUserStatus,     // 유저 상태 조회
}

// 게임 관련 이벤트 핸들러
var gameEvents = map[EventType]ExecutionEvent{
	EventGameStart:  HandleGameStart,  // 게임 시작
	EventGameEnd:    HandleGameEnd,    // 게임 종료
	EventGameAction: HandleGameAction, // 플레이어 행동
	EventGameSync:   HandleGameSync,   // 게임 상태 동기화
	EventGamePause:  HandleGamePause,  // 게임 일시정지
	EventGameInfo:   HandleGameInfo,   // 게임 설명 출력
}

// 채팅 관련 이벤트 핸들러
var chatEvents = map[EventType]ExecutionEvent{
	EventChatSend:    HandleChatSend,    // 채팅 메시지 전송
	EventChatHistory: HandleChatHistory, // 채팅 내역 조회
	EventChatMute:    HandleChatMute,    // 유저 채팅 제한
}

// 시스템 관련 이벤트 핸들러
var systemEvents = map[EventType]ExecutionEvent{
	EventSystemPing:   HandleSystemPing,   // 핑 체크
	EventSystemError:  HandleSystemError,  // 에러 전달
	EventSystemNotice: HandleSystemNotice, // 시스템 공지
	EventSystemSync:   HandleSystemSync,   // 시스템 전체 상태 동기화
}

func sendResult(u *user.Session, eventType EventType, data interface{}, resultMsgCode string) {
	if u.Conn == nil {
		return
	}
	res := createWebSocketResult(eventType, data, resultMsgCode, "ko")
	u.WriteMutex.Lock()
	defer u.WriteMutex.Unlock()
	_ = u.Conn.WriteJSON(res)
}
func sendError(u *user.Session, resultMsgCode string) {
	if u.Conn == nil {
		return
	}
	res := createWebSocketResult(EventError, nil, resultMsgCode, "ko")
	u.WriteMutex.Lock()
	defer u.WriteMutex.Unlock()
	_ = u.Conn.WriteJSON(res)
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
