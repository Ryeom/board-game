package ws

import (
	"context"
	"fmt"
	"github.com/Ryeom/board-game/room"
	"github.com/Ryeom/board-game/user"
	"time"
)

func HandleDefault(ctx context.Context, user *user.Session, event SocketEvent) {
	// TODO logging
	user.Conn.WriteJSON(map[string]any{
		"type":    "error",
		"message": fmt.Sprintf("unknown event type: %s", event.Type),
	})
}

// HandleRoomCreate 방 생성하기
func HandleRoomCreate(ctx context.Context, user *user.Session, event SocketEvent) {
	rid := "room:" + user.ID + ":" + fmt.Sprint(time.Now().UnixNano())
	r := &room.Room{
		ID:        rid,
		Host:      user.ID,
		Players:   []string{user.ID},
		GameMode:  room.GameModeHanabi,
		CreatedAt: time.Now(),
	}
	r.Save()
	user.Conn.WriteJSON(map[string]any{
		"type": "room.create",
		"data": map[string]string{
			"roomId": r.ID,
		},
	})

	rooms := room.ListRooms(ctx)
	user.Conn.WriteJSON(map[string]any{
		"type": "room.list",
		"data": rooms,
	})
}

// HandleRoomJoin 방에 참여하기
func HandleRoomJoin(ctx context.Context, user *user.Session, event SocketEvent) {
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
		"type": "room.join",
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

// HandleRoomLeave 방 나가기
func HandleRoomLeave(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleRoomList 현재 방 조회
func HandleRoomList(ctx context.Context, user *user.Session, _ SocketEvent) {
	rooms := room.ListRooms(ctx)
	user.Conn.WriteJSON(map[string]any{
		"type": "room.list",
		"data": rooms,
	})
}

// HandleRoomUpdate 방 설정 변경
func HandleRoomUpdate(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleRoomDelete 방 삭제 (방에 아무도 없으면 삭제)
func HandleRoomDelete(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleRoomKick 방에서 퇴장
func HandleRoomKick(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleUserIdentify 유저 초기 식별
func HandleUserIdentify(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleUserUpdate 유저 정보 업데이트
func HandleUserUpdate(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleUserDisconnect 유저 연결 종료
func HandleUserDisconnect(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleUserStatus 유저 상태 조회
func HandleUserStatus(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleGameStart 게임 시작
func HandleGameStart(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleGameEnd 게임 종료
func HandleGameEnd(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleGameAction 플레이어 행동
func HandleGameAction(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleGameSync 게임 상태 동기화
func HandleGameSync(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleGamePause 게임 일시정지
func HandleGamePause(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleGameInfo 게임 설명 출력
func HandleGameInfo(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleChatSend 채팅 메시지 전송
func HandleChatSend(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleChatHistory 채팅 내역 조회
func HandleChatHistory(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleChatMute 유저 채팅 제한
func HandleChatMute(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleSystemPing 핑 체크
func HandleSystemPing(ctx context.Context, user *user.Session, event SocketEvent) {
	user.Conn.WriteJSON(map[string]string{
		"type":    "pong",
		"message": "pong",
	})
}

// HandleSystemError 에러 전달
func HandleSystemError(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleSystemNotice 시스템 공지
func HandleSystemNotice(ctx context.Context, user *user.Session, event SocketEvent) {}

// HandleSystemSync 시스템 전체 상태 동기화
func HandleSystemSync(ctx context.Context, user *user.Session, event SocketEvent) {}
