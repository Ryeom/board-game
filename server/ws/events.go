package ws

import (
	"context"
	"encoding/json"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	ae "github.com/Ryeom/board-game/internal/errors"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
	"time"
)

func HandleDefault(ctx context.Context, u *user.Session, event SocketEvent) {
	sendError(u, ae.BadRequest(fmt.Sprintf("알 수 없는 이벤트 타입: %s", event.Type), nil))
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
	// 기존 로직 유지. 방 생성 시에는 아직 Set에 추가할 필요 없음 (입장 시 추가)
	rid := "room:" + u.ID + ":" + fmt.Sprint(time.Now().UnixNano())
	r := &room.Room{
		ID:        rid,
		Host:      u.ID,
		Players:   []string{u.ID},
		GameMode:  room.GameModeHanabi,
		CreatedAt: time.Now(),
	}
	r.Save() // 방 정보 저장

	// 방 생성 시 방장은 자동으로 방에 참여하므로, Set에 추가
	if err := redisutil.AddSet(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), u.ID); err != nil {
		log.Logger.Errorf("HandleRoomCreate - Failed to add host to room sessions set: %v", err)
		sendError(u, ae.InternalServerError("방 생성 중 오류가 발생했습니다.", err))
		return
	}

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
		sendError(u, ae.NotFound("방을 찾을 수 없습니다.", nil))
		return
	}

	// 사용자의 RoomID 업데이트
	oldRoomID := u.RoomID // 기존 방 ID 저장
	u.RoomID = r.ID
	// 변경된 세션 정보를 Redis에 저장 (RoomID가 업데이트됨)
	_ = user.SaveUserSession(u)

	joined, err := r.Join(ctx, u.ID)
	if err != nil {
		log.Logger.Errorf("HandleRoomJoin - Room join error: %v", err)
		sendError(u, ae.InternalServerError("방 참여 실패", err))
		return
	}
	if !joined { // 이미 참여한 경우
		sendError(u, ae.Conflict("이미 방에 참여 중입니다.", nil))
		return
	}

	// 이전 방이 있었다면 해당 방의 Set에서 제거
	if oldRoomID != "" && oldRoomID != r.ID {
		_ = redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(oldRoomID), u.ID)
	}

	// 새 방의 세션 Set에 사용자 ID 추가
	if err := redisutil.AddSet(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), u.ID); err != nil {
		log.Logger.Errorf("HandleRoomJoin - Failed to add user %s to room %s sessions set: %v", u.ID, r.ID, err)
		// 에러 처리 또는 알림
	}

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
		sendError(u, ae.BadRequest("방에 참여한 상태가 아닙니다.", nil))
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, ae.NotFound("방을 찾을 수 없습니다.", nil))
		return
	}

	// Redis Set에서 플레이어 제거
	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), u.ID); err != nil {
		log.Logger.Errorf("HandleRoomLeave - Failed to remove user %s from room %s sessions set: %v", u.ID, r.ID, err)
		// 에러 처리
	}

	// 플레이어 제거 (기존 Room 로직)
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
		// 방 삭제 시 해당 방의 세션 Set도 삭제 (선택 사항, 필요시 구현)
		err := redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID))
		if err != nil {
			log.Logger.Errorf("HandleRoomLeave - Failed to delete room %s sessions set: %v", r.ID, err)
		}
	} else {
		_ = r.Save()
	}

	// 유저의 RoomID 초기화 및 세션 정보 업데이트
	u.RoomID = ""
	_ = user.SaveUserSession(u)

	// 본인에게 알림
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
		sendError(u, ae.NotFound("방에 참여한 상태가 아닙니다.", nil))
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, ae.NotFound("방을 찾을 수 없습니다.", nil))
		return
	}

	if r.Host != u.ID {
		sendError(u, ae.Unauthorized("방장에게만 권한이 있습니다.", nil))
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
		sendError(u, ae.InternalServerError("변경 저장 시 오류가 발생했습니다.", nil))
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
		sendError(u, ae.NotFound("지정된 사용자가 없습니다.", nil))
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, ae.NotFound("방 정보를 찾을 수 없습니다.", nil))
		return
	}

	if r.Host != u.ID {
		sendError(u, ae.Unauthorized("강제 퇴장 권한이 없습니다.", nil))
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
		sendError(u, ae.NotFound("당 유저는 방에 없습니다.", nil))
		return
	}

	r.Players = newPlayers
	if err := r.Save(); err != nil {
		sendError(u, ae.InternalServerError("방 정보를 저장하는 데 실패했습니다.", nil))
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
		sendError(u, ae.InternalServerError("필수값이 없습니다.", nil))
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
		sendError(u, ae.NotFound("변경할 항목이 없습니다.", nil))
		return
	}

	sendResult(u, "user.update", updated, "유저 정보가 업데이트되었습니다.")
}

// HandleUserDisconnect 유저 연결 종료
func HandleUserDisconnect(ctx context.Context, u *user.Session, event SocketEvent) {
	roomID := u.RoomID
	if roomID == "" {
		// 방에 속해있지 않으면 Redis Set에서 제거할 필요 없음
		return
	}

	// 방에 속해 있었다면 해당 방의 Set에서 제거
	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(roomID), u.ID); err != nil {
		log.Logger.Errorf("HandleUserDisconnect - Failed to remove user %s from room %s sessions set: %v", u.ID, roomID, err)
		// 에러 처리
	}

	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return // 방이 이미 사라졌을 수 있음
	}

	// 해당 유저 제거 (기존 Room 로직)
	updatedPlayers := make([]string, 0, len(r.Players))
	for _, pid := range r.Players {
		if pid != u.ID {
			updatedPlayers = append(updatedPlayers, pid)
		}
	}
	r.Players = updatedPlayers

	if len(r.Players) == 0 {
		_ = room.DeleteRoom(ctx, r.ID)
		// 방 삭제 시 해당 방의 세션 Set도 삭제 (선택 사항, 필요시 구현)
		_ = redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID))
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

	// 유저의 RoomID 초기화 및 상태 업데이트
	u.RoomID = ""
	u.Status = "disconnected"
	// Redis에서 세션 정보를 완전히 삭제
	_ = user.DeleteUserSession(u.ID)
}

// HandleUserStatus 유저 상태 조회
func HandleUserStatus(ctx context.Context, u *user.Session, event SocketEvent) {
	targetID, ok := event.Data["userId"].(string)
	if !ok || targetID == "" {
		sendError(u, ae.NotFound("필수값이 부족합니다.", nil))
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
