package ws

import (
	"context"
	"github.com/Ryeom/board-game/user"
)

func dispatchSocketEvent(ctx context.Context, user *user.Session, event SocketEvent) {
	handler := getHandler(event.Type)
	handler(ctx, user, event)
}

func getHandler(eventType string) EventHandler {
	if handler, ok := eventHandlers[eventType]; ok {
		return handler
	}
	return HandleDefault
}

type EventHandler func(ctx context.Context, user *user.Session, event SocketEvent)

func mergeHandlers(maps ...map[string]EventHandler) map[string]EventHandler {
	merged := make(map[string]EventHandler)
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
var roomEvents = map[string]EventHandler{
	"room.create": HandleRoomCreate, // 방 생성
	"room.join":   HandleRoomJoin,   // 방 참가
	"room.leave":  HandleRoomLeave,  // 방 나가기
	"room.list":   HandleRoomList,   // 방 목록 조회
	"room.update": HandleRoomUpdate, // 방 설정 변경
	"room.delete": HandleRoomDelete, // 방 삭제
	"room.kick":   HandleRoomKick,   // 강제 퇴장
}

// 유저 관련 이벤트 핸들러
var userEvents = map[string]EventHandler{
	"user.identify":   HandleUserIdentify,   // 유저 초기 식별
	"user.update":     HandleUserUpdate,     // 유저 정보 업데이트
	"user.disconnect": HandleUserDisconnect, // 유저 연결 종료
	"user.status":     HandleUserStatus,     // 유저 상태 조회
}

// 게임 관련 이벤트 핸들러
var gameEvents = map[string]EventHandler{
	"game.start":  HandleGameStart,  // 게임 시작
	"game.end":    HandleGameEnd,    // 게임 종료
	"game.action": HandleGameAction, // 플레이어 행동
	"game.sync":   HandleGameSync,   // 게임 상태 동기화
	"game.pause":  HandleGamePause,  // 게임 일시정지
	"game.info":   HandleGameInfo,   // 게임 설명 출력
}

// 채팅 관련 이벤트 핸들러
var chatEvents = map[string]EventHandler{
	"chat.send":    HandleChatSend,    // 채팅 메시지 전송
	"chat.history": HandleChatHistory, // 채팅 내역 조회
	"chat.mute":    HandleChatMute,    // 유저 채팅 제한
}

// 시스템 관련 이벤트 핸들러
var systemEvents = map[string]EventHandler{
	"system.ping":   HandleSystemPing,   // 핑 체크
	"system.error":  HandleSystemError,  // 에러 전달
	"system.notice": HandleSystemNotice, // 시스템 공지
	"system.sync":   HandleSystemSync,   // 시스템 전체 상태 동기화
}
