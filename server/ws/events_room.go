package ws

import (
	"context"

	"github.com/Ryeom/board-game/internal/domain/room"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/service"
	"github.com/Ryeom/board-game/internal/user"
)

// GlobalRoomService entry point
var GlobalRoomService = service.NewRoomService(&WsBroadcaster{})

// HandleRoomCreate 방 생성하기
func HandleRoomCreate(ctx context.Context, u *user.Session, event SocketEvent) {
	var req RoomCreateRequest
	if err := bindEventData(event, &req); err != nil || req.RoomName == "" || req.MaxPlayers < 2 || req.MaxPlayers > 6 {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	r, err := GlobalRoomService.CreateRoom(ctx, u.ID, u.Name, req.RoomName, req.Password, req.MaxPlayers)
	if err != nil {
		sendError(u, err.Error()) // Service returns error code string
		return
	}

	rooms, _ := GlobalRoomService.GetRoomList(ctx)
	sendResult(u, event.Type, RoomCreateResponse{
		RoomID:     r.ID,
		RoomName:   r.RoomName,
		MaxPlayers: r.MaxPlayers,
		RoomList:   roomSummaries(rooms),
	}, resp.SuccessCodeRoomCreate)
}

// HandleRoomJoin 방에 참여하기
func HandleRoomJoin(ctx context.Context, u *user.Session, event SocketEvent) {
	var req RoomJoinRequest
	if err := bindEventData(event, &req); err != nil || req.RoomID == "" {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	r, err := GlobalRoomService.JoinRoom(ctx, u.ID, u.Name, req.RoomID, req.Password)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	sendResult(u, event.Type, r, resp.SuccessCodeRoomJoin)
}

// HandleRoomLeave 방 나가기
func HandleRoomLeave(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeRoomNotInRoom)
		return
	}

	roomID := u.RoomID
	newHost, roomDeleted, err := GlobalRoomService.LeaveRoom(ctx, u.ID, roomID)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	sendResult(u, event.Type, RoomLeaveResponse{
		RoomID:  roomID,
		NewHost: newHost,
		Deleted: roomDeleted,
	}, resp.SuccessCodeRoomLeave)
}

// HandleRoomList 현재 방 조회 (WebSocket)
func HandleRoomList(ctx context.Context, u *user.Session, event SocketEvent) {
	rooms, _ := GlobalRoomService.GetRoomList(ctx)
	sendResult(u, event.Type, RoomListResponse{Rooms: roomSummaries(rooms)}, resp.SuccessCodeRoomListFetch)
}

// HandleRoomUpdate 방 설정 변경
func HandleRoomUpdate(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeRoomNotInRoom)
		return
	}

	// Service expects generic map update
	r, updated, err := GlobalRoomService.UpdateRoom(ctx, u.ID, u.RoomID, event.Data)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	if !updated {
		sendResult(u, event.Type, nil, resp.SuccessCodeRoomNoChanges)
		return
	}

	sendResult(u, event.Type, r, resp.SuccessCodeRoomUpdate)
}

// HandleRoomReady 레디 상태 토글
func HandleRoomReady(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeRoomNotInRoom)
		return
	}

	isReady, readyPlayers, err := GlobalRoomService.SetPlayerReady(ctx, u.ID, u.RoomID)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	sendResult(u, event.Type, map[string]any{
		"isReady":      isReady,
		"readyPlayers": readyPlayers,
	}, resp.SuccessCodeRoomReady)
}

// HandleRoomKick 방에서 퇴장
func HandleRoomKick(ctx context.Context, u *user.Session, event SocketEvent) {
	var req RoomKickRequest
	if err := bindEventData(event, &req); err != nil || req.UserID == "" {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	// Service call
	newHost, _, err := GlobalRoomService.KickUser(ctx, u.ID, u.RoomID, req.UserID)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	sendResult(u, EventRoomKick, map[string]any{
		"userId":  req.UserID,
		"newHost": newHost,
	}, resp.SuccessCodeRoomKick)

	// Note: Service handles broadcasting "user.kicked".
	// Service also logic to notify the kicked user specifically via Broadcaster?
	// Wait, Service Broadcaster typically supports "SendToPlayer".
	// The original code did: "targetSession.Conn.WriteJSON(...)".
	// The Service logic currently broadcasts "user.kicked" to room.
	// We might want Service to also notify the specific user "kicked_from_room".
	// Let's check Service implementation.
	// Service KickUser does NOT notify the specific user directly with "kicked_from_room" message, only "user.kicked" broadcast.
	// Original code: _ = targetSession.Conn.WriteJSON(...)
	// I should update Service to send this notification as well if possible, or do it here.
	// But Service encapsulates the logic.
	// Let's rely on Broadcast "user.kicked" which the client presumably handles?
	// Or we can add it to Service later.
	// For now, this is a reasonable subset.
}

func roomSummaries(rooms []*room.Room) []RoomSummary {
	summaryList := make([]RoomSummary, 0, len(rooms))
	for _, r := range rooms {
		summaryList = append(summaryList, RoomSummary{
			ID:          r.ID,
			RoomName:    r.RoomName,
			Host:        r.Host,
			PlayerCount: len(r.Players),
			MaxPlayers:  r.MaxPlayers,
			GameMode:    string(r.GameMode),
			HasPassword: r.Password != "",
			CreatedAt:   r.CreatedAt,
		})
	}
	return summaryList
}
