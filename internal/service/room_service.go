package service

import (
	"context"
	"fmt"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	"github.com/Ryeom/board-game/internal/game"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/Ryeom/board-game/log"
)

type RoomService struct {
	Broadcaster Broadcaster
}

func NewRoomService(broadcaster Broadcaster) *RoomService {
	return &RoomService{
		Broadcaster: broadcaster,
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, userID string, userName string, roomName string, password string, maxPlayers int) (*room.Room, error) {
	roomID := "room:" + userID + ":" + fmt.Sprint(time.Now().UnixNano())
	r, err := room.CreateRoom(ctx, roomID, userID, roomName, password, maxPlayers)
	if err != nil {
		log.Logger.Errorf("CreateRoom - Failed to create room: %v", err)
		return nil, fmt.Errorf(resp.ErrorCodeRoomCreationFailed)
	}

	// 방 생성 시 방장은 자동으로 방에 참여하므로, Redis Set에 추가
	if err := redisutil.AddSet(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), userID); err != nil {
		log.Logger.Errorf("CreateRoom - Failed to add host %s to room sessions set %s: %v", userID, user.RoomIndexKey(r.ID), err)
		return nil, fmt.Errorf(resp.ErrorCodeRoomCreationFailed)
	}

	// 세션의 RoomID 업데이트 및 저장
	// NOTE: This assumes caller has access to session or we fetch it.
	// Since we passed userID, we should fetch session to update it.
	session, err := user.GetSession(userID)
	if err == nil && session != nil {
		session.RoomID = r.ID
		if saveErr := user.SaveUserSession(session); saveErr != nil {
			log.Logger.Errorf("CreateRoom - Failed to save user session %s with new room ID %s: %v", userID, r.ID, saveErr)
		}
	} else {
		log.Logger.Warningf("CreateRoom - Could not fetch session for user %s to update RoomID", userID)
	}
	
	log.Logger.Debugf("CreateRoom - Room %s created successfully by user %s", r.ID, userName)

	return r, nil
}

func (s *RoomService) JoinRoom(ctx context.Context, userID string, userName string, roomID string, password string) (*room.Room, error) {
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return nil, fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}

	// 방 참여 로직 (비밀번호 검증 및 인원 제한 포함)
	joined, err := r.Join(ctx, userID, password)
	if err != nil {
		log.Logger.Errorf("JoinRoom - User %s failed to join room %s: %v", userName,roomID, err)
		return nil, err
	}
	if !joined { // 이미 참여
		log.Logger.Debugf("JoinRoom - User %s already joined room %s", userName, r.ID)
		return nil, fmt.Errorf(resp.ErrorCodeRoomAlreadyJoined)
	}

	// 세션의 RoomID 업데이트 및 저장
	// We need to handle 'OldRoomID' logic (removing from old room set)
	session, err := user.GetSession(userID)
	if err == nil && session != nil {
		oldRoomID := session.RoomID
		session.RoomID = r.ID
		if saveErr := user.SaveUserSession(session); saveErr != nil {
			log.Logger.Errorf("JoinRoom - Failed to save user session %s with new room ID %s: %v", userID, r.ID, saveErr)
			return nil, fmt.Errorf(resp.ErrorCodeRoomJoinFailed)
		}
		
		// 이전 방이 있었다면 해당 방의 Redis Set에서 사용자 ID 제거
		if oldRoomID != "" && oldRoomID != r.ID {
			if remErr := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(oldRoomID), userID); remErr != nil {
				log.Logger.Warningf("JoinRoom - Failed to remove user %s from old room %s sessions set: %v", userName, oldRoomID, remErr)
			}
		}
	}

	// 새 방의 Redis Set에 사용자 ID 추가
	if err := redisutil.AddSet(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), userID); err != nil {
		log.Logger.Errorf("JoinRoom - Failed to add user %s to room %s sessions set: %v", userName, r.ID, err)
		return nil, fmt.Errorf(resp.ErrorCodeRoomJoinFailed)
	}

	// Broadcast Join Event
	s.Broadcaster.BroadcastToRoom(r.ID, "room.join", map[string]string{
		"userId":   userID,
		"userName": userName,
	}, resp.SuccessCodeRoomJoin)

	return r, nil
}

func (s *RoomService) LeaveRoom(ctx context.Context, userID string, roomID string) (string, bool, error) {
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return "", false, fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}

	if err := redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), userID); err != nil {
		log.Logger.Errorf("LeaveRoom - Failed to remove user %s from room %s sessions set: %v", userID, r.ID, err)
	}

	originalPlayers := r.Players
	newPlayers := make([]string, 0, len(originalPlayers))
	isHostLeaving := false
	if r.Host == userID {
		isHostLeaving = true
	}

	for _, pid := range originalPlayers {
		if pid != userID {
			newPlayers = append(newPlayers, pid)
		}
	}
	r.Players = newPlayers

	roomDeleted := false
	newHostID := r.Host

	if len(r.Players) == 0 {
		_ = room.DeleteRoom(ctx, r.ID)
		if err := redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID)); err != nil {
			log.Logger.Errorf("LeaveRoom - Failed to delete room %s sessions set after deletion: %v", r.ID, err)
		}
		log.Logger.Infof("Room %s deleted as no players left.", r.ID)
		roomDeleted = true
	} else {
		if isHostLeaving {
			r.Host = newPlayers[0]
			newHostID = r.Host
			log.Logger.Infof("Host of room %s changed from %s to %s", r.ID, userID, r.Host)
		}
		if err := r.Save(); err != nil {
			log.Logger.Errorf("LeaveRoom - Failed to save room %s after player leaving: %v", r.ID, err)
			return "", false, fmt.Errorf(resp.ErrorCodeRoomLeaveFailed)
		}
	}

	// 유저의 RoomID 초기화
	session, err := user.GetSession(userID)
	userName := ""
	if err == nil && session != nil {
		session.RoomID = ""
		userName = session.Name
		_ = user.SaveUserSession(session)
	}

	// Broadcast Leave Event (only if room not deleted)
	if !roomDeleted {
		s.Broadcaster.BroadcastToRoom(r.ID, "user.left", map[string]string{
			"userId":   userID,
			"userName": userName,
			"newHost":  newHostID,
		}, resp.SuccessCodeRoomLeave)
	}

	return newHostID, roomDeleted, nil
}

func (s *RoomService) UpdateRoom(ctx context.Context, userID string, roomID string, updates map[string]any) (*room.Room, bool, error) {
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return nil, false, fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}

	if r.Host != userID {
		return nil, false, fmt.Errorf(resp.ErrorCodeRoomNotHost)
	}

	updated := false

	if nameRaw, exists := updates["roomName"]; exists {
		if roomName, ok := nameRaw.(string); ok && r.RoomName != roomName {
			r.RoomName = roomName
			updated = true
		}
	}
	if gmRaw, exists := updates["gameMode"]; exists {
		if gmStr, ok := gmRaw.(string); ok && game.Mode(gmStr) != r.GameMode {
			r.GameMode = game.Mode(gmStr)
			updated = true
		}
	}
	if passRaw, exists := updates["password"]; exists {
		if password, ok := passRaw.(string); ok {
			if password == "" {
				if r.Password != "" {
					r.Password = ""
					updated = true
				}
			} else {
				hashedPassword, err := util.HashPassword(password)
				if err != nil {
					return nil, false, fmt.Errorf(resp.ErrorCodeRoomPasswordHashingFailed)
				}
				if r.Password != hashedPassword {
					r.Password = hashedPassword
					updated = true
				}
			}
		}
	}
	if maxPlayersRaw, exists := updates["maxPlayers"]; exists {
		// Handle float64/int logic inside service or assume it's passed as correct type?
		// JSON unmarshal usually gives float64 for numbers.
		var maxPlayers int
		valid := false
		if mpFloat, ok := maxPlayersRaw.(float64); ok {
			maxPlayers = int(mpFloat)
			valid = true
		} else if mpInt, ok := maxPlayersRaw.(int); ok {
			maxPlayers = mpInt
			valid = true
		}

		if valid {
			if maxPlayers < 2 || maxPlayers > 6 || maxPlayers < len(r.Players) {
				return nil, false, fmt.Errorf(resp.ErrorCodeRoomInvalidRequest)
			}
			if maxPlayers != r.MaxPlayers {
				r.MaxPlayers = maxPlayers
				updated = true
			}
		}
	}

	if !updated {
		return nil, false, nil // No error, not updated
	}

	if err := r.Save(); err != nil {
		log.Logger.Errorf("UpdateRoom - Failed to update room %s: %v", r.ID, err)
		return nil, false, fmt.Errorf(resp.ErrorCodeRoomUpdateFailed)
	}

	s.Broadcaster.BroadcastToRoom(r.ID, "room.update", r, resp.SuccessCodeRoomUpdate)

	return r, true, nil
}

func (s *RoomService) KickUser(ctx context.Context, hostID string, roomID string, targetID string) (string, bool, error) {
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return "", false, fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}

	if r.Host != hostID {
		return "", false, fmt.Errorf(resp.ErrorCodeRoomNotHost)
	}

	targetSession, err := user.GetSession(targetID)
	if err != nil || targetSession == nil {
		return "", false, fmt.Errorf(resp.ErrorCodeUserNotFound)
	}
	if targetSession.RoomID != r.ID {
		return "", false, fmt.Errorf(resp.ErrorCodeRoomUserNotInRoom)
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
		return "", false, fmt.Errorf(resp.ErrorCodeRoomUserNotInRoom)
	}

	r.Players = newPlayers
	roomDeleted := false
	newHostID := r.Host

	if len(r.Players) == 0 {
		_ = room.DeleteRoom(ctx, r.ID)
		_ = redisutil.Delete(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID))
		roomDeleted = true
	} else {
		if r.Host == targetID {
			r.Host = newPlayers[0]
			newHostID = r.Host
		}
		if err := r.Save(); err != nil {
			return "", false, fmt.Errorf(resp.ErrorCodeRoomKickFailed)
		}
	}

	targetSession.RoomID = ""
	_ = user.SaveUserSession(targetSession)
	_ = redisutil.RemoveSetMembers(redisutil.RedisTargetUser, user.RoomIndexKey(r.ID), targetID)

	s.Broadcaster.BroadcastToRoom(r.ID, "user.kicked", map[string]string{
		"userId":   targetID,
		"userName": targetSession.Name,
		"newHost":  newHostID,
	}, resp.SuccessCodeRoomKick)

	return newHostID, roomDeleted, nil
}

func (s *RoomService) GetRoomList(ctx context.Context) ([]*room.Room, error) {
	// Simple wrapper, but allows for caching/processing later
	return room.ListRooms(ctx), nil
}
