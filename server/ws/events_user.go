package ws

import (
	"context"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	log "github.com/Ryeom/board-game/log"
)

// HandleUserIdentify 유저 초기 식별
func HandleUserIdentify(ctx context.Context, u *user.Session, event SocketEvent) {
	data := event.Data
	userID, ok1 := data["userId"].(string)
	userName, ok2 := data["userName"].(string)

	if !ok1 || !ok2 || userID == "" || userName == "" {
		sendError(u, resp.ErrorCodeAuthInvalidRequest)
		return
	}

	u.ID = userID
	u.Name = userName

	// 유저 정보 Redis에 캐싱
	// SaveUserSession은 ctx를 받지 않으므로, 직접 호출
	if err := user.SaveUserSession(u); err != nil {
		log.Logger.Errorf("HandleUserIdentify - Failed to save user session %s after identify: %v", u.ID, err)
		sendError(u, resp.ErrorCodeWSInitialSessionSaveFailed)
		return
	}

	sendResult(u, "user.identify", map[string]string{
		"userId":   userID,
		"userName": userName,
	}, resp.SuccessCodeUserIdentify)
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
		sendError(u, resp.ErrorCodeUserNoUpdates)
		return
	}
	// 세션 정보 업데이트 (Redis)
	if err := user.SaveUserSession(u); err != nil {
		log.Logger.Errorf("HandleUserUpdate - Failed to save user session %s after update: %v", u.ID, err)
		sendError(u, resp.ErrorCodeUserProfileUpdateFailed)
		return
	}

	sendResult(u, "user.update", updated, resp.SuccessCodeUserUpdate)
}

// HandleUserDisconnect 유저 연결 종료
func HandleUserDisconnect(ctx context.Context, u *user.Session, event SocketEvent) {
	roomID := u.RoomID
	if roomID == "" { // 방에 속해있지 않으면 Redis Set에서 제거할 필요 없음
		if err := user.DeleteUserSession(u.ID); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to delete user session %s (no room): %v", u.ID, err)
		}
		return
	}

	// 방에 속해 있었다면 해당 방의 Set에서 제거
	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(roomID), u.ID); err != nil {
		log.Logger.Errorf("HandleUserDisconnect - Failed to remove user %s from room %s sessions set: %v", u.ID, roomID, err)
		// 에러 처리
	}

	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		log.Logger.Warningf("HandleUserDisconnect - Room %s not found for disconnected user %s. Cleaning up session.", roomID, u.ID)
		if err := user.DeleteUserSession(u.ID); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to delete user session %s (room not found case): %v", u.ID, err)
		}
		return // 방이 이미 사라졌다면 더 이상 방 관련 업데이트 불필요
	}

	updatedPlayers := make([]string, 0, len(r.Players))
	for _, pid := range r.Players {
		if pid != u.ID {
			updatedPlayers = append(updatedPlayers, pid)
		}
	}
	r.Players = updatedPlayers

	if len(r.Players) == 0 { // 유저 연결 끊김으로 방이 비면 삭제
		_ = room.DeleteRoom(ctx, r.ID)
		if err := redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID)); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to delete room %s sessions set after disconnect deletion: %v", r.ID, err)
		}
		log.Logger.Infof("Room %s deleted as no players left after disconnect.", r.ID)
	} else { // 방에 플레이어가 남아있다면
		if r.Host == u.ID { // 방장이 연결 끊김으로 나갔다면 새로운 방장 위임
			r.Host = updatedPlayers[0] // 첫 번째 남은 플레이어를 새 방장으로 위임
			log.Logger.Infof("Host of room %s changed from %s to %s due to disconnect.", r.ID, u.ID, r.Host)
		}
		if err := r.Save(); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to save room %s after disconnect: %v", r.ID, err)
			// 이 에러는 클라이언트에게 알리지 않고 로깅만
		}
		GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
			"type": "user.left",
			"data": map[string]string{
				"userId":   u.ID,
				"userName": u.Name,
				"newHost":  r.Host, // 새 방장 정보도 함께 전달
			},
		})
	}

	u.RoomID = ""
	u.Status = "disconnected"
	if err := user.DeleteUserSession(u.ID); err != nil {
		log.Logger.Errorf("HandleUserDisconnect - Failed to delete user session %s: %v", u.ID, err)
	}
}

// HandleUserStatus 유저 상태 조회
func HandleUserStatus(ctx context.Context, u *user.Session, event SocketEvent) {
	targetID, ok := event.Data["userId"].(string)
	if !ok || targetID == "" {
		sendError(u, resp.ErrorCodeUserInvalidRequest)
		return
	}

	target, err := user.GetSession(targetID)
	if err != nil { // 세션이 없으면 오프라인으로 간주
		sendResult(u, event.Type, map[string]any{
			"online": false,
		}, resp.ErrorCodeUserNotFound)
		return
	}

	sendResult(u, event.Type, map[string]any{
		"online": true,
		"userId": target.ID,
		"name":   target.Name,
		"roomId": target.RoomID,
		"status": target.Status,
	}, resp.SuccessCodeUserStatusFetch)
}
