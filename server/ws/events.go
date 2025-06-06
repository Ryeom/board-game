package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/room"
	"github.com/Ryeom/board-game/user"
	"time"
)

func HandleDefault(ctx context.Context, u *user.Session, event SocketEvent) {
	sendError(u, event.Type, fmt.Sprintf("unknown event type: %s", event.Type))
	eventJSON, _ := json.Marshal(event)
	log.Logger.Warningf(
		"[UNKNOWN_EVENT] type=%s userID=%s userName=%s roomID=%s ip=%s ua=%s event=%s",
		event.Type,
		u.ID,
		u.Name,
		u.RoomID,
		u.IP,
		u.UserAgent,
		string(eventJSON),
	)
}

// HandleRoomCreate 방 생성하기
func HandleRoomCreate(ctx context.Context, u *user.Session, event SocketEvent) {
	rid := "room:" + u.ID + ":" + fmt.Sprint(time.Now().UnixNano())
	r := &room.Room{
		ID:        rid,
		Host:      u.ID,
		Players:   []string{u.ID},
		GameMode:  room.GameModeHanabi,
		CreatedAt: time.Now(),
	}
	r.Save()
	rooms := room.ListRooms(ctx)
	sendResult(u, event.Type, map[string]interface{}{
		"room_id":   r.ID,
		"room_list": rooms,
	}, "")
}

// HandleRoomJoin 방에 참여하기
func HandleRoomJoin(ctx context.Context, u *user.Session, event SocketEvent) {
	r, ok := room.GetRoom(ctx, event.RoomID)
	if !ok {
		sendError(u, event.Type, "room not found")
		return
	}
	u.RoomID = r.ID
	_, _ = r.Join(ctx, u.ID)
	u.Conn.WriteJSON(map[string]any{
		"type": "room.join",
		"data": r,
	})
	sendResult(u, event.Type, map[string]any{
		"type": "room.join",
		"data": r,
	}, "")
	GlobalBroadcaster.BroadcastToRoom(u.RoomID, map[string]any{
		"type": "room.join",
		"data": map[string]string{
			"userId":   u.ID,
			"userName": u.Name,
		},
	})
}

// HandleRoomLeave 방 나가기
func HandleRoomLeave(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, event.Type, "no room to leave")
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, event.Type, "room not found")
		return
	}

	// 플레이어 제거
	newPlayers := make([]string, 0, len(r.Players))
	for _, pid := range r.Players {
		if pid != u.ID {
			newPlayers = append(newPlayers, pid)
		}
	}
	r.Players = newPlayers

	// 방에 아무도 없으면 삭제
	if len(r.Players) == 0 {
		_ = room.DeleteRoom(ctx, r.ID)
	} else {
		_ = r.Save()
	}

	// 유저의 RoomID 초기화
	u.RoomID = ""

	// 본인에게 알림)
	sendResult(u, event.Type, map[string]any{
		"type": "room_left",
		"data": map[string]string{
			"roomId": r.ID,
		},
	}, "")

	// 나머지 인원에게 알림
	GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
		"type": "user_left",
		"data": map[string]string{
			"userId": u.ID,
			"name":   u.Name,
		},
	})
}

// HandleRoomList 현재 방 조회
func HandleRoomList(ctx context.Context, u *user.Session, event SocketEvent) {
	rooms := room.ListRooms(ctx)
	sendResult(u, event.Type, map[string]any{
		"type": "room.list",
		"data": rooms,
	}, "")
}

// HandleRoomUpdate 방 설정 변경
// HandleRoomUpdate 방 설정 변경
func HandleRoomUpdate(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, event.Type, "no room to update")
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, event.Type, "room not found")
		return
	}

	if r.Host != u.ID {
		sendError(u, event.Type, "only host can update room settings")
		return
	}

	// 필드 업데이트 처리 (예: gameMode 변경)
	if gmRaw, exists := event.Data["gameMode"]; exists {
		if gmStr, ok := gmRaw.(string); ok {
			r.GameMode = room.GameMode(gmStr)
		}
	}

	// 변경 저장
	if err := r.Save(); err != nil {
		sendError(u, event.Type, "failed to update room")
		return
	}

	// 본인에게 알림
	u.Conn.WriteJSON(map[string]any{
		"type": "room_updated",
		"data": r,
	})

	// 다른 인원에게도 알림
	GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
		"type": "room_updated",
		"data": r,
	})
}

// HandleRoomDelete 방 삭제 (방에 아무도 없으면 삭제)
//func HandleRoomDelete(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleRoomKick 방에서 퇴장
func HandleRoomKick(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleUserIdentify 유저 초기 식별
func HandleUserIdentify(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleUserUpdate 유저 정보 업데이트
func HandleUserUpdate(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleUserDisconnect 유저 연결 종료
func HandleUserDisconnect(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleUserStatus 유저 상태 조회
func HandleUserStatus(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleGameStart 게임 시작
func HandleGameStart(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleGameEnd 게임 종료
func HandleGameEnd(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleGameAction 플레이어 행동
func HandleGameAction(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleGameSync 게임 상태 동기화
func HandleGameSync(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleGamePause 게임 일시정지
func HandleGamePause(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleGameInfo 게임 설명 출력
func HandleGameInfo(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleChatSend 채팅 메시지 전송
func HandleChatSend(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleChatHistory 채팅 내역 조회
func HandleChatHistory(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleChatMute 유저 채팅 제한
func HandleChatMute(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleSystemPing 핑 체크
func HandleSystemPing(ctx context.Context, u *user.Session, event SocketEvent) {
	u.Conn.WriteJSON(map[string]string{
		"type":    "pong",
		"message": "pong",
	})
}

// HandleSystemError 에러 전달
func HandleSystemError(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleSystemNotice 시스템 공지
func HandleSystemNotice(ctx context.Context, u *user.Session, event SocketEvent) {}

// HandleSystemSync 시스템 전체 상태 동기화
func HandleSystemSync(ctx context.Context, u *user.Session, event SocketEvent) {}
