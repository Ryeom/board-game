package ws

import (
	"context"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	log "github.com/Ryeom/board-game/log"
	"time"
)

// HandleUserIdentify 유저 초기 식별 (재접속 포함)
func HandleUserIdentify(ctx context.Context, u *user.Session, event SocketEvent) {
	data := event.Data
	requestedUserID, ok1 := data["userId"].(string)
	userName, ok2 := data["userName"].(string)

	if !ok1 || !ok2 || requestedUserID == "" || userName == "" {
		sendError(u, resp.ErrorCodeAuthInvalidRequest)
		return
	}

	oldSessionID := u.ID

	// 기존 세션 확인 (재접속 감지)
	prevSession, err := user.GetSession(requestedUserID)
	if err == nil && prevSession != nil && prevSession.Status == "disconnected" && prevSession.RoomID != "" {
		// 재접속: 기존 세션 정보 복원하되 새 연결 사용
		u.ID = prevSession.ID
		u.Name = prevSession.Name
		u.RoomID = prevSession.RoomID
		u.IsHost = prevSession.IsHost
		u.Status = "connected"

		ActiveSessions().Delete(oldSessionID)
		ActiveSessions().Store(u.ID, u)

		if err := user.SaveUserSession(u); err != nil {
			log.Logger.Errorf("HandleUserIdentify - Failed to save reconnected session %s: %v", u.ID, err)
			sendError(u, resp.ErrorCodeWSInitialSessionSaveFailed)
			return
		}

		// room_sessions set에 다시 추가 (브로드캐스터가 찾을 수 있도록)
		if err := redisutil.AddSet(redisutil.RedisTargetUser, user.RoomIndexKey(u.RoomID), u.ID); err != nil {
			log.Logger.Errorf("HandleUserIdentify - Failed to re-add user %s to room sessions set: %v", u.ID, err)
		}

		log.Logger.Infof("[Reconnected] ID: %s | Name: %s | Room: %s | Time: %s",
			u.ID, u.Name, u.RoomID, time.Now().Format(time.RFC3339))

		// 방에 재접속 알림
		GlobalBroadcaster.BroadcastToRoom(u.RoomID, map[string]any{
			"type": "user.reconnected",
			"data": map[string]string{
				"userId":   u.ID,
				"userName": u.Name,
			},
		})

		// 게임 진행 중이면 게임 상태 전송
		r, roomOk := room.GetRoom(ctx, u.RoomID)
		if roomOk && r.IsGameStarted {
			state, gameMode, gsErr := GlobalGameService.GetGameState(ctx, u.RoomID, u.ID)
			if gsErr == nil {
				sendResult(u, "game.sync", map[string]any{
					"roomId":      u.RoomID,
					"gameMode":    gameMode,
					"gameState":   state,
					"reconnected": true,
				}, resp.SuccessCodeGameSync)
			}
		}

		sendResult(u, "user.identify", map[string]any{
			"userId":      u.ID,
			"userName":    u.Name,
			"roomId":      u.RoomID,
			"reconnected": true,
		}, resp.SuccessCodeUserReconnected)
		return
	}

	// 일반 식별 (신규 접속)
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

	log.Logger.Infof("[Identified] ID: %s | Name: %s | IP: %s | Time: %s",
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

	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		log.Logger.Warningf("HandleUserDisconnect - Room %s not found for user %s. Cleaning up session.", u.RoomID, u.ID)
		if err := user.DeleteUserSession(u.ID); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to delete user session %s (room not found): %v", u.ID, err)
		}
		if u.Conn != nil {
			_ = u.Conn.Close()
		}
		return
	}

	// 게임 진행 중이면 세션 보존 (재접속 대기)
	if r.IsGameStarted {
		u.Status = "disconnected"
		u.Conn = nil
		if err := user.SaveUserSession(u); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to save disconnected session %s: %v", u.ID, err)
		}
		log.Logger.Infof("Player %s disconnected during game in room %s. Session preserved for reconnection.", u.ID, r.ID)

		GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
			"type": "user.disconnected",
			"data": map[string]string{
				"userId":   u.ID,
				"userName": u.Name,
			},
		})
		return
	}

	// 게임 중이 아니면 기존 로직: 방에서 제거, 세션 삭제
	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(roomID), u.ID); err != nil {
		log.Logger.Errorf("HandleUserDisconnect - Failed to remove user %s from room %s sessions set: %v", u.ID, roomID, err)
	}

	updatedPlayers := make([]string, 0, len(r.Players))
	for _, pid := range r.Players {
		if pid != u.ID {
			updatedPlayers = append(updatedPlayers, pid)
		}
	}
	r.Players = updatedPlayers
	r.ResetReady()

	if len(r.Players) == 0 {
		_ = room.DeleteRoom(ctx, r.ID)
		if err := redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID)); err != nil {
			log.Logger.Errorf("HandleUserDisconnect - Failed to delete room %s sessions set: %v", r.ID, err)
		}
		log.Logger.Infof("Room %s deleted as no players left after disconnect.", r.ID)
	} else {
		if r.Host == u.ID {
			r.Host = updatedPlayers[0]
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
