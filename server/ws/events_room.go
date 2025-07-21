package ws

import (
	"context"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	"github.com/Ryeom/board-game/internal/game"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/internal/util"
	log "github.com/Ryeom/board-game/log"
	"time"
)

// HandleRoomCreate 방 생성하기
func HandleRoomCreate(ctx context.Context, u *user.Session, event SocketEvent) {
	// 1. 요청 데이터 파싱 및 유효성 검사
	roomName, ok := event.Data["roomName"].(string)
	if !ok || roomName == "" {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}
	password, _ := event.Data["password"].(string)
	maxPlayersFloat, ok := event.Data["maxPlayers"].(float64)
	maxPlayers := int(maxPlayersFloat)
	if !ok || maxPlayers < 2 || maxPlayers > 6 {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	// 2. 방 ID 생성 및 방 생성 (room.CreateRoom 함수에 인자 추가)
	roomID := "room:" + u.ID + ":" + fmt.Sprint(time.Now().UnixNano())
	r, err := room.CreateRoom(ctx, roomID, u.ID, roomName, password, maxPlayers)
	if err != nil {
		log.Logger.Errorf("HandleRoomCreate - Failed to create room: %v", err)
		sendError(u, resp.ErrorCodeRoomCreationFailed)
		return
	}

	// 3. 방 생성 시 방장은 자동으로 방에 참여하므로, Redis Set에 추가
	if err := redisutil.AddSet(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), u.ID); err != nil {
		log.Logger.Errorf("HandleRoomCreate - Failed to add host %s to room sessions set %s: %v", u.ID, user.RoomIndexKey(r.ID), err) // 로그 추가
		sendError(u, resp.ErrorCodeRoomCreationFailed)
		return
	}

	// 4. 세션의 RoomID 업데이트 및 저장 (선택 사항: HandleUserIdentify에서 처리하면 여기서는 안해도 됨)
	u.RoomID = r.ID
	if err := user.SaveUserSession(u); err != nil {
		log.Logger.Errorf("HandleRoomCreate - Failed to save user session %s with new room ID %s: %v", u.RoomID, err)
	}

	// 5. 방 목록 갱신 및 응답
	rooms := room.ListRooms(ctx)
	sendResult(u, event.Type, map[string]interface{}{
		"room_id":     r.ID,
		"room_name":   r.RoomName,
		"max_players": r.MaxPlayers,
		"room_list":   rooms,
	}, resp.SuccessCodeRoomCreate)
	log.Logger.Debugf("HandleRoomCreate - Room %s created successfully by user %s (ActualUserID: %s)", r.ID, u.Name, u.ActualUserID) // 로그 추가
}

// HandleRoomJoin 방에 참여하기
func HandleRoomJoin(ctx context.Context, u *user.Session, event SocketEvent) {
	log.Logger.Debugf("HandleRoomJoin - User %s (ActualUserID: %s) attempting to join room. Event: %+v", u.Name, u.ActualUserID, event) // Debug log

	// 1. 요청 데이터 파싱
	roomID, ok := event.Data["roomId"].(string)
	if !ok || roomID == "" {
		log.Logger.Debugf("HandleRoomJoin - Invalid room ID in request from user %s", u.Name)
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}
	password, _ := event.Data["password"].(string) // 선택사항

	// 2. 방 존재 여부 확인
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		log.Logger.Debugf("HandleRoomJoin - Room %s not found for user %s", roomID, u.Name)
		sendError(u, resp.ErrorCodeRoomNotFound)
		return
	}

	log.Logger.Debugf("HandleRoomJoin - Room %s fetched. Current players: %d, Max players: %d, IsGameStarted: %t, HasPassword: %t",
		r.ID, len(r.Players), r.MaxPlayers, r.IsGameStarted, r.Password != "")

	// 3. 방 참여 로직 (비밀번호 검증 및 인원 제한 포함)
	joined, err := r.Join(ctx, u.ID, password)
	if err != nil {
		log.Logger.Errorf("HandleRoomJoin - User %s (socketId: %s) failed to join room %s. Error from r.Join: %v", u.Name, u.ID, r.ID, err) // Log exact error from r.Join
		sendError(u, err.Error())                                                                                                           // err.Error() will return the error string (e.g., "ERROR_ROOM_FULL")
		return
	}
	if !joined { // 이미 참여
		log.Logger.Debugf("HandleRoomJoin - User %s (socketId: %s) already joined room %s", u.Name, u.ID, r.ID) // Debug log
		sendError(u, resp.ErrorCodeRoomAlreadyJoined)
		return
	}

	// 4. 세션의 RoomID 업데이트 및 저장
	oldRoomID := u.RoomID
	u.RoomID = r.ID
	if err := user.SaveUserSession(u); err != nil {
		log.Logger.Errorf("HandleRoomJoin - Failed to save user session %s (socketId: %s) with new room ID %s: %v", u.Name, u.ID, u.RoomID, err)
		sendError(u, resp.ErrorCodeRoomJoinFailed)
		return
	}

	// 5. 이전 방이 있었다면 해당 방의 Redis Set에서 사용자 ID 제거
	if oldRoomID != "" && oldRoomID != r.ID {
		if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(oldRoomID), u.ID); err != nil {
			log.Logger.Warningf("HandleRoomJoin - Failed to remove user %s (socketId: %s) from old room %s sessions set: %v", u.Name, u.ID, oldRoomID, err)
		}
	}

	// 6. 새 방의 Redis Set에 사용자 ID 추가
	if err := redisutil.AddSet(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), u.ID); err != nil {
		log.Logger.Errorf("HandleRoomJoin - Failed to add user %s (socketId: %s) to room %s sessions set: %v", u.Name, u.ID, r.ID, err)
		sendError(u, resp.ErrorCodeRoomJoinFailed)
		return
	}

	log.Logger.Debugf("HandleRoomJoin - User %s (ActualUserID: %s) successfully joined room %s. Room now has %d players.", u.Name, u.ActualUserID, r.ID, len(r.Players)) // Success log

	// 7. 클라이언트 응답 및 브로드캐스트
	sendResult(u, event.Type, r, resp.SuccessCodeRoomJoin)
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
		sendError(u, resp.ErrorCodeRoomNotInRoom)
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		log.Logger.Warningf("HandleRoomLeave - Room %s not found for user %s trying to leave. Cleaning up session.", u.RoomID, u.ID)
		u.RoomID = ""
		_ = user.SaveUserSession(u)
		sendError(u, resp.ErrorCodeRoomNotFound)
		return
	}

	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), u.ID); err != nil {
		log.Logger.Errorf("HandleRoomLeave - Failed to remove user %s from room %s sessions set: %v", u.ID, r.ID, err)
	}

	originalPlayers := r.Players
	newPlayers := make([]string, 0, len(originalPlayers))
	isHostLeaving := false
	if r.Host == u.ID { // 방장이 나가는 경우
		isHostLeaving = true
	}

	for _, pid := range originalPlayers {
		if pid != u.ID {
			newPlayers = append(newPlayers, pid)
		}
	}
	r.Players = newPlayers

	// 방에 아무도 없으면 삭제
	if len(r.Players) == 0 {
		_ = room.DeleteRoom(ctx, r.ID)
		// 방 삭제 시 해당 방의 세션 Set도 삭제
		if err := redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID)); err != nil {
			log.Logger.Errorf("HandleRoomLeave - Failed to delete room %s sessions set after room deletion: %v", r.ID, err)
		}
		log.Logger.Infof("Room %s deleted as no players left.", r.ID)
	} else { // 방에 플레이어가 남아있다면
		if isHostLeaving { // 방장이 나갔다면 새로운 방장 위임
			r.Host = newPlayers[0] // 첫 번째 남은 플레이어를 새 방장으로 위임 (더 복잡한 로직 가능)
			log.Logger.Infof("Host of room %s changed from %s to %s", r.ID, u.ID, r.Host)
		}
		if err := r.Save(); err != nil {
			log.Logger.Errorf("HandleRoomLeave - Failed to save room %s after player leaving: %v", r.ID, err)
			sendError(u, resp.ErrorCodeRoomLeaveFailed)
			return
		}
	}

	// 유저의 RoomID 초기화 및 세션 정보 업데이트
	u.RoomID = ""
	if err := user.SaveUserSession(u); err != nil {
		log.Logger.Errorf("HandleRoomLeave - Failed to save user session %s after leaving room: %v", u.ID, err)
	}

	// 본인에게 알림
	sendResult(u, event.Type, map[string]any{
		"type":    "room_left",
		"roomId":  r.ID,
		"newHost": r.Host,
	}, resp.SuccessCodeRoomLeave)

	// 나머지 인원에게 알림 (방이 삭제되지 않은 경우에만)
	if len(r.Players) > 0 {
		GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
			"type": "user.left",
			"data": map[string]string{
				"userId":   u.ID,
				"userName": u.Name,
				"newHost":  r.Host,
			},
		})
	}
}

// HandleRoomList 현재 방 조회 (WebSocket)
func HandleRoomList(ctx context.Context, u *user.Session, event SocketEvent) {
	rooms := room.ListRooms(ctx)

	type roomSummary struct {
		ID          string    `json:"id"`
		RoomName    string    `json:"roomName"`
		Host        string    `json:"host"`
		PlayerNum   int       `json:"playerCount"`
		MaxPlayers  int       `json:"maxPlayers"`
		GameMode    string    `json:"gameMode"`
		HasPassword bool      `json:"hasPassword"`
		CreatedAt   time.Time `json:"createdAt"`
	}

	summaryList := make([]roomSummary, 0, len(rooms))
	for _, r := range rooms {
		summaryList = append(summaryList, roomSummary{
			ID:          r.ID,
			RoomName:    r.RoomName,
			Host:        r.Host,
			PlayerNum:   len(r.Players),
			MaxPlayers:  r.MaxPlayers,
			GameMode:    string(r.GameMode),
			HasPassword: r.Password != "",
			CreatedAt:   r.CreatedAt,
		})
	}

	sendResult(u, event.Type, map[string]any{
		"type": "room.list",
		"data": summaryList,
	}, resp.SuccessCodeRoomListFetch)
}

// HandleRoomUpdate 방 설정 변경
func HandleRoomUpdate(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeRoomNotInRoom)
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, resp.ErrorCodeRoomNotFound)
		return
	}

	if r.Host != u.ID {
		sendError(u, resp.ErrorCodeRoomNotHost)
		return
	}

	// 1. 요청 데이터 파싱
	updated := false

	if nameRaw, exists := event.Data["roomName"]; exists {
		if roomName, ok := nameRaw.(string); ok && r.RoomName != roomName {
			r.RoomName = roomName
			updated = true
		}
	}
	if gmRaw, exists := event.Data["gameMode"]; exists {
		if gmStr, ok := gmRaw.(string); ok && game.Mode(gmStr) != r.GameMode {
			// TODO: 지원하는 게임 모드인지 추가 검증 필요
			r.GameMode = game.Mode(gmStr)
			updated = true
		}
	}
	// 비밀번호 업데이트 (새 비밀번호가 제공된 경우 해싱하여 저장)
	if passRaw, exists := event.Data["password"]; exists {
		if password, ok := passRaw.(string); ok {
			if password == "" {
				if r.Password != "" {
					r.Password = ""
					updated = true
				}
			} else {
				hashedPassword, err := util.HashPassword(password)
				if err != nil {
					log.Logger.Errorf("HandleRoomUpdate - Password Hashing Error: %v", err)
					sendError(u, resp.ErrorCodeRoomPasswordHashingFailed)
					return
				}
				// 기존과 다를 경우만 업데이트
				if r.Password != hashedPassword {
					r.Password = hashedPassword
					updated = true
				}
			}
		}
	}
	// 최대 플레이어 수 업데이트
	if maxPlayersRaw, exists := event.Data["maxPlayers"]; exists {
		if maxPlayersFloat, ok := maxPlayersRaw.(float64); ok {
			maxPlayers := int(maxPlayersFloat)
			if maxPlayers < 2 { // 최소 인원 제한
				sendError(u, resp.ErrorCodeRoomInvalidRequest)
				return
			}
			if maxPlayers > 6 { // 최대 인원 제한
				sendError(u, resp.ErrorCodeRoomInvalidRequest)
				return
			}
			if maxPlayers < len(r.Players) { // 현재 플레이어 수보다 낮게 설정 불가
				sendError(u, resp.ErrorCodeRoomInvalidRequest)
				return
			}
			if maxPlayers != r.MaxPlayers {
				r.MaxPlayers = maxPlayers
				updated = true
			}
		}
	}

	if !updated {
		sendResult(u, event.Type, nil, resp.SuccessCodeRoomNoChanges)
		return
	}

	// 변경 저장
	if err := r.Save(); err != nil {
		log.Logger.Errorf("HandleRoomUpdate - Failed to update room %s: %v", r.ID, err)
		sendError(u, resp.ErrorCodeRoomUpdateFailed)
		return
	}

	sendResult(u, event.Type, r, resp.SuccessCodeRoomUpdate)
	GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
		"type": "room.update",
		"data": r,
	})
}

// HandleRoomKick 방에서 퇴장
func HandleRoomKick(ctx context.Context, u *user.Session, event SocketEvent) {
	targetID, ok := event.Data["userId"].(string)
	if !ok || targetID == "" {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, resp.ErrorCodeRoomNotFound)
		return
	}

	if r.Host != u.ID {
		sendError(u, resp.ErrorCodeRoomNotHost)
		return
	}

	// 강퇴 대상 세션을 Redis에서 조회
	targetSession, err := user.GetSession(targetID)
	if err != nil || targetSession == nil {
		sendError(u, resp.ErrorCodeUserNotFound)
		return
	}
	if targetSession.RoomID != r.ID { // 대상 유저가 현재 방에 속해있는지 확인
		sendError(u, resp.ErrorCodeRoomUserNotInRoom)
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
		sendError(u, resp.ErrorCodeRoomUserNotInRoom)
		return
	}

	r.Players = newPlayers
	if len(r.Players) == 0 { // 유저 강퇴로 방이 비면 삭제
		_ = room.DeleteRoom(ctx, r.ID)
		if err := redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID)); err != nil {
			log.Logger.Errorf("HandleRoomKick - Failed to delete room %s sessions set after kick deletion: %v", r.ID, err)
		}
		log.Logger.Infof("Room %s deleted as no players left after kick.", r.ID)
	} else { // 방에 플레이어가 남아있다면
		if r.Host == targetID { // 방장이 강퇴당했다면 새로운 방장 위임
			r.Host = newPlayers[0] // 첫 번째 남은 플레이어를 새 방장으로 위임
			log.Logger.Infof("Host of room %s changed from %s to %s due to kick.", r.ID, targetID, r.Host)
		}
		if err := r.Save(); err != nil {
			log.Logger.Errorf("HandleRoomKick - Failed to save room %s after kick: %v", r.ID, err)
			sendError(u, resp.ErrorCodeRoomKickFailed)
			return
		}
	}

	// 강퇴 대상 유저의 RoomID 초기화 및 세션 업데이트
	targetSession.RoomID = ""
	if err := user.SaveUserSession(targetSession); err != nil {
		log.Logger.Errorf("HandleRoomKick - Failed to save kicked user session %s: %v", targetID, err)
	}
	// Redis Set에서 강퇴 대상 제거
	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), targetID); err != nil {
		log.Logger.Errorf("HandleRoomKick - Failed to remove kicked user %s from room %s sessions set: %v", targetID, r.ID, err)
	}

	sendResult(u, "room.kick", map[string]any{
		"userId":  targetID,
		"newHost": r.Host, // 새 방장 정보도 함께 전달
	}, resp.SuccessCodeRoomKick) // 성공 메시지

	if targetSession.Conn != nil { // 강퇴된 유저에게도 알림 전송 (연결이 살아있다면)
		_ = targetSession.Conn.WriteJSON(map[string]any{
			"type":    "kicked_from_room",
			"message": fmt.Sprintf("방장 %s 에 의해 방 %s 에서 강퇴되었습니다.", u.Name, r.ID),
			"roomId":  r.ID,
		})
	}

	GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
		"type": "user.kicked",
		"data": map[string]string{
			"userId":   targetID,
			"userName": targetSession.Name,
			"newHost":  r.Host,
		},
	})
}
