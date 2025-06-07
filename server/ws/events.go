package ws

import (
	"context"
	"encoding/json"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
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
func HandleRoomKick(ctx context.Context, u *user.Session, event SocketEvent) {
	targetID, ok := event.Data["userId"].(string)
	if !ok || targetID == "" {
		sendError(u, "room.kick", "userId가 필요합니다.")
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, "room.kick", "방 정보를 찾을 수 없습니다.")
		return
	}

	if r.Host != u.ID {
		sendError(u, "room.kick", "방장만 강퇴할 수 있습니다.")
		return
	}

	kicked := false
	newPlayers := make([]string, 0, len(r.Players))
	for _, pid := range r.Players {
		if pid == targetID {
			kicked = true
			continue
		}
		newPlayers = append(newPlayers, pid)
	}
	if !kicked {
		sendError(u, "room.kick", "해당 유저는 방에 없습니다.")
		return
	}

	r.Players = newPlayers
	if err := r.Save(); err != nil {
		sendError(u, "room.kick", "방 정보를 저장하는 데 실패했습니다.")
		return
	}

	sendResult(u, "room.kick", map[string]any{
		"userId": targetID,
	}, "유저가 강퇴되었습니다.")

	// TODO: 실제 대상 유저가 접속 중이라면 그쪽에도 알림 전송 필요
}

// HandleUserIdentify 유저 초기 식별
func HandleUserIdentify(ctx context.Context, u *user.Session, event SocketEvent) {
	data := event.Data
	userID, ok1 := data["userId"].(string)
	userName, ok2 := data["userName"].(string)

	if !ok1 || !ok2 || userID == "" || userName == "" {
		sendError(u, "user.identify", "userId와 userName이 필요합니다.")
		return
	}

	u.ID = userID
	u.Name = userName

	// 유저 정보 Redis에 캐싱
	redisutil.SaveJSON("user", "user:"+userID, u, time.Hour)

	sendResult(u, "user.identify", map[string]string{
		"userId":   userID,
		"userName": userName,
	}, "유저 식별 완료")
}

// HandleUserUpdate 유저 정보 업데이트
func HandleUserUpdate(ctx context.Context, u *user.Session, event SocketEvent) {
	data := event.Data
	updated := map[string]string{}

	if name, ok := data["name"].(string); ok && name != "" {
		u.Name = name
		updated["name"] = name
	}
	if status, ok := data["status"].(string); ok {
		u.Status = status
		updated["status"] = status
	}

	if len(updated) == 0 {
		sendError(u, "user.update", "변경할 항목이 없습니다.")
		return
	}

	sendResult(u, "user.update", updated, "유저 정보가 업데이트되었습니다.")
}

// HandleUserDisconnect 유저 연결 종료
func HandleUserDisconnect(ctx context.Context, u *user.Session, event SocketEvent) {
	roomID := u.RoomID
	if roomID == "" {
		return // 방에 속해있지 않으면 아무것도 안 함
	}

	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return
	}

	// 해당 유저 제거
	updated := []string{}
	for _, pid := range r.Players {
		if pid != u.ID {
			updated = append(updated, pid)
		}
	}
	r.Players = updated

	if len(r.Players) == 0 {
		_ = room.DeleteRoom(ctx, r.ID)
	} else {
		_ = r.Save()
		// 나머지 유저에게 알림
		GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
			"type": "user.left",
			"data": map[string]string{
				"userId": u.ID,
				"name":   u.Name,
			},
		})
	}

	// 로그 또는 상태 초기화
	u.RoomID = ""
	u.Status = "disconnected"
}

// HandleUserStatus 유저 상태 조회
func HandleUserStatus(ctx context.Context, u *user.Session, event SocketEvent) {
	targetID, ok := event.Data["userId"].(string)
	if !ok || targetID == "" {
		sendError(u, event.Type, "조회할 userId가 없습니다.")
		return
	}

	target, err := user.GetSession(targetID) // 세션 관리자가 userID로 조회 가능해야 함
	if err != nil {
		sendResult(u, event.Type, map[string]any{
			"online": false,
		}, "사용자가 오프라인 상태입니다.")
		return
	}

	// 접속 중인 유저 정보 반환
	sendResult(u, event.Type, map[string]any{
		"online": true,
		"userId": target.ID,
		"name":   target.Name,
		"roomId": target.RoomID,
		"status": target.Status,
	}, "유저 상태 조회 성공")
}

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
