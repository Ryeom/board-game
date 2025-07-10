package ws

import (
	"context"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	log "github.com/Ryeom/board-game/log"
	"time"
)

// HandleUserIdentify 유저 초기 식별
func HandleUserIdentify(ctx context.Context, u *user.Session, event SocketEvent) {
	data := event.Data
	requestedUserID, ok1 := data["userId"].(string)
	userName, ok2 := data["userName"].(string)

	if !ok1 || !ok2 || requestedUserID == "" || userName == "" {
		sendError(u, resp.ErrorCodeAuthInvalidRequest)
		return
	}

	oldSessionID := u.ID

	u.ID = requestedUserID
	u.Name = userName
	u.Status = "connected"

	if oldSessionID != requestedUserID {
		ActiveSessions().Delete(oldSessionID)
		ActiveSessions().Store(u.ID, u)
	} else {
		ActiveSessions().Store(u.ID, u)
	}

	if err := user.SaveUserSession(u); err != nil {
		log.Logger.Errorf("HandleUserIdentify - Failed to save user session %s after identify: %v", u.ID, err)
		sendError(u, resp.ErrorCodeWSInitialSessionSaveFailed)
		return
	}

	fmt.Printf(
		"[Identified] ID: %s | Name: %s | IP: %s | Time: %s\n",
		u.ID, u.Name, u.IP, u.ConnectedAt.Format(time.RFC3339),
	)

	sendResult(u, "user.identify", map[string]string{
		"userId":   u.ID,
		"userName": u.Name,
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

	if err := user.SaveUserSession(u); err != nil {
		log.Logger.Errorf("HandleUserUpdate - Failed to save user session %s after update: %v", u.ID, err)
		sendError(u, resp.ErrorCodeUserProfileUpdateFailed)
		return
	}

	sendResult(u, "user.update", updated, resp.SuccessCodeUserUpdate)
}

// HandleUserDisconnect 유저 연결 종료
func HandleUserDisconnect(ctx context.Context, u *user.Session, event SocketEvent) {
	// Remove from activeSessions map first
	ActiveSessions().Delete(u.ID)

	roomID := u.RoomID
	if roomID == "" {
		if err := user.DeleteUserSession(u.ID); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to delete user session %s (no room): %v", u.ID, err)
		}
		if u.Conn != nil {
			_ = u.Conn.Close()
		}
		return
	}

	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(roomID), u.ID); err != nil {
		log.Logger.Errorf("HandleUserDisconnect - Failed to remove user %s from room %s sessions set: %v", u.ID, roomID, err)
		// 에러 처리
	}

	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		log.Logger.Warningf("HandleUserDisconnect - Room %s not found for user %s trying to leave. Cleaning up session.", u.RoomID, u.ID)
		if err := user.DeleteUserSession(u.ID); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to delete user session %s (room not found case): %v", u.ID, err)
		}
		if u.Conn != nil {
			_ = u.Conn.Close()
		}
		return
	}

	updatedPlayers := make([]string, 0, len(r.Players))
	for _, pid := range r.Players {
		if pid != u.ID {
			updatedPlayers = append(updatedPlayers, pid)
		}
	}
	r.Players = updatedPlayers

	if len(r.Players) == 0 {
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
		}
		GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
			"type": "user.left",
			"data": map[string]string{
				"userId":   u.ID,
				"userName": u.Name,
				"newHost":  r.Host,
			},
		})
	}

	u.RoomID = ""
	u.Status = "disconnected"
	if err := user.DeleteUserSession(u.ID); err != nil {
		log.Logger.Errorf("HandleUserDisconnect - Failed to delete user session %s: %v", u.ID, err)
	}
	if u.Conn != nil {
		_ = u.Conn.Close()
	}
}

// HandleUserStatus 유저 상태 조회
func HandleUserStatus(ctx context.Context, u *user.Session, event SocketEvent) {
	targetID, ok := event.Data["userId"].(string)
	if !ok || targetID == "" {
		sendError(u, resp.ErrorCodeUserInvalidRequest)
		return
	}

	val, found := ActiveSessions().Load(targetID)
	if !found {
		sendResult(u, event.Type, map[string]any{
			"online": false,
		}, resp.ErrorCodeUserNotFound)
		return
	}

	target, ok := val.(*user.Session)
	if !ok {
		log.Logger.Errorf("HandleUserStatus: Found non-session type in activeSessions for ID %s", targetID)
		sendError(u, resp.ErrorCodeUserProfileFetchFailed)
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
