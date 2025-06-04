package ws

import (
	"context"
	"fmt"
	"github.com/Ryeom/board-game/room"
	"log"
	"time"
)

// 방 관련 이벤트 핸들러
var roomEvents = map[string]EventHandler{
	"room.create": HandleDefault, // 방 생성
	"room.join":   HandleDefault, // 방 참가
	"room.leave":  HandleDefault, // 방 나가기
	"room.list":   HandleDefault, // 방 목록 조회
	"room.update": HandleDefault, // 방 설정 변경
	"room.delete": HandleDefault, // 방 삭제
	"room.kick":   HandleDefault, // 강제 퇴장
}

// 유저 관련 이벤트 핸들러
var userEvents = map[string]EventHandler{
	"user.identify":   HandleDefault, // 유저 초기 식별
	"user.update":     HandleDefault, // 유저 정보 업데이트
	"user.disconnect": HandleDefault, // 유저 연결 종료
	"user.status":     HandleDefault, // 유저 상태 조회
}

// 게임 관련 이벤트 핸들러
var gameEvents = map[string]EventHandler{
	"game.start":  HandleDefault, // 게임 시작
	"game.end":    HandleDefault, // 게임 종료
	"game.action": HandleDefault, // 플레이어 행동
	"game.sync":   HandleDefault, // 게임 상태 동기화
	"game.pause":  HandleDefault, // 게임 일시정지
}

// 채팅 관련 이벤트 핸들러
var chatEvents = map[string]EventHandler{
	"chat.send":    HandleDefault, // 채팅 메시지 전송
	"chat.history": HandleDefault, // 채팅 내역 조회
	"chat.mute":    HandleDefault, // 유저 채팅 제한
}

// 시스템 관련 이벤트 핸들러
var systemEvents = map[string]EventHandler{
	"system.ping":   HandleDefault, // 핑 체크
	"system.error":  HandleDefault, // 에러 전달
	"system.notice": HandleDefault, // 시스템 공지
	"system.sync":   HandleDefault, // 시스템 전체 상태 동기화
}
var eventHandlers = mergeHandlers(
	roomEvents,
	userEvents,
	gameEvents,
	chatEvents,
	systemEvents,
)

func dispatchSocketEvent(user *UserSession, event SocketEvent) {
	ctx := context.Background()

	if handler, ok := eventHandlers[event.Type]; ok {
		handler(ctx, user, event)
	} else {
		log.Println("⚠️ Unknown event type:", event.Type)
	}
}

type EventHandler func(ctx context.Context, user *UserSession, event SocketEvent)

func mergeHandlers(maps ...map[string]EventHandler) map[string]EventHandler {
	merged := make(map[string]EventHandler)
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}
func HandleDefault(ctx context.Context, user *UserSession, event SocketEvent) {}

func HandleGameInfo(ctx context.Context, user *UserSession, event SocketEvent) {
	gameInfo := []string{}
	user.Conn.WriteJSON(map[string]any{
		"type": "room_list",
		"data": gameInfo,
	})
}
func HandleRoomCreate(ctx context.Context, user *UserSession, event SocketEvent) {
	rid := "room:" + user.ID + ":" + fmt.Sprint(time.Now().UnixNano())
	r := &room.Room{
		ID:        rid,
		Host:      user.ID,
		Players:   []string{user.ID},
		GameMode:  room.GameModeHanabi,
		CreatedAt: time.Now(),
	}
	r.Save(ctx)
	user.Conn.WriteJSON(map[string]any{
		"type": "room_created",
		"data": map[string]string{
			"roomId": r.ID,
		},
	})

	rooms := room.ListRooms(ctx)
	user.Conn.WriteJSON(map[string]any{
		"type": "room_list",
		"data": rooms,
	})
}

// Used in eventHandlers["room.join"]
func HandleRoomJoin(ctx context.Context, user *UserSession, event SocketEvent) {
	r, ok := room.GetRoom(ctx, event.RoomID)
	if !ok {
		user.Conn.WriteJSON(map[string]any{
			"type":    "error",
			"message": "room not found",
		})
		return
	}
	user.RoomID = r.ID
	_, _ = r.Join(ctx, user.ID)
	user.Conn.WriteJSON(map[string]any{
		"type": "room_joined",
		"data": r,
	})
	//GlobalBroadcaster.BroadcastToRoom(user.RoomID, map[string]any{
	//	"type": "user_joined",
	//	"data": map[string]string{
	//		"userId":   user.ID,
	//		"userName": user.Name,
	//	},
	//})
}

// Used in eventHandlers["room.list"]
func HandleRoomList(ctx context.Context, user *UserSession, _ SocketEvent) {
	rooms := room.ListRooms(ctx)
	user.Conn.WriteJSON(map[string]any{
		"type": "room_list",
		"data": rooms,
	})
}

// Used in eventHandlers["room.start"] -- placeholder for now
func HandleRoomStart(ctx context.Context, user *UserSession, event SocketEvent) {
	fmt.Println("🎮 start game in room:", event.RoomID)
	// TODO: Start game logic
}
