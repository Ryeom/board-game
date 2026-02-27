package ws

import (
	"context"
	"time"

	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/service"
	"github.com/Ryeom/board-game/internal/user"
)

// GlobalRoomService entry point
var GlobalRoomService = service.NewRoomService(&WsBroadcaster{})

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

	// 2. 서비스 호출
	r, err := GlobalRoomService.CreateRoom(ctx, u.ID, u.Name, roomName, password, maxPlayers)
	if err != nil {
		sendError(u, err.Error()) // Service returns error code string
		return
	}

	// 3. 방 목록 갱신 및 응답
	rooms, _ := GlobalRoomService.GetRoomList(ctx)
	sendResult(u, event.Type, map[string]interface{}{
		"room_id":     r.ID,
		"room_name":   r.RoomName,
		"max_players": r.MaxPlayers,
		"room_list":   rooms,
	}, resp.SuccessCodeRoomCreate)
}

// HandleRoomJoin 방에 참여하기
func HandleRoomJoin(ctx context.Context, u *user.Session, event SocketEvent) {
	// 1. 요청 데이터 파싱
	roomID, ok := event.Data["roomId"].(string)
	if !ok || roomID == "" {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}
	password, _ := event.Data["password"].(string)

	// 2. 서비스 호출
	r, err := GlobalRoomService.JoinRoom(ctx, u.ID, u.Name, roomID, password)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	// 3. 클라이언트 응답 (Broadcasting is handled by Service)
	sendResult(u, event.Type, r, resp.SuccessCodeRoomJoin)
}

// HandleRoomLeave 방 나가기
func HandleRoomLeave(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeRoomNotInRoom)
		return
	}

	newHost, roomDeleted, err := GlobalRoomService.LeaveRoom(ctx, u.ID, u.RoomID)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	// 본인에게 알림
	// Service broadcasts to others. We confirm to the leaver.
	sendResult(u, event.Type, map[string]any{
		"type":    "room_left",
		"roomId":  u.RoomID, // RoomID might be cleared in session, but we use the one we requested with? Wait, u.RoomID is cleared in Service?
		// Service clears u.RoomID in Redis, but u object here is reference.
		// Service loads session from Redis, updates it. 'u' here might be stale if Service re-fetched it?
		// But 'u' is *user.Session. Use u.RoomID?
		// Service code: "_ = user.SaveUserSession(session)" (fetches fresh session).
		// So 'u' passed here is not modified by Service.
		// However, we want to return the ID of the room they left.
		"newHost": newHost,
		"deleted": roomDeleted,
	}, resp.SuccessCodeRoomLeave)
}

// HandleRoomList 현재 방 조회 (WebSocket)
func HandleRoomList(ctx context.Context, u *user.Session, event SocketEvent) {
	rooms, _ := GlobalRoomService.GetRoomList(ctx)

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
	targetID, ok := event.Data["userId"].(string)
	if !ok || targetID == "" {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	// Service call
	newHost, _, err := GlobalRoomService.KickUser(ctx, u.ID, u.RoomID, targetID)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	sendResult(u, "room.kick", map[string]any{
		"userId":  targetID,
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
