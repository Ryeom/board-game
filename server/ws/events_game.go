package ws

import (
	"context"

	"github.com/Ryeom/board-game/internal/game"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/service"
	"github.com/Ryeom/board-game/internal/user"
)

// WsBroadcaster implements service.Broadcaster using the WebSocket GlobalBroadcaster
type WsBroadcaster struct{}

func (b *WsBroadcaster) SendToPlayer(playerID string, eventName string, payload any, msgCode string) {
	res := createWebSocketResult(eventName, payload, msgCode, "ko")
	GlobalBroadcaster.SendToPlayer(playerID, res)
}

func (b *WsBroadcaster) BroadcastToRoom(roomID string, eventName string, payload any, msgCode string) {
	res := createWebSocketResult(eventName, payload, msgCode, "ko")
	GlobalBroadcaster.BroadcastToRoom(roomID, res)
}

// GlobalGameService is the entry point for all game logic
var GlobalGameService = service.NewGameService(game.NewManager(), &WsBroadcaster{})

// HandleGameStart (game.start)게임 시작
func HandleGameStart(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
		return
	}

	err := GlobalGameService.StartGame(ctx, u.RoomID, u.ID)
	if err != nil {
		sendError(u, err.Error())
		return
	}
	// Success notification is handled by the Service through Broadcaster
}

// HandleGameEnd 게임 종료
func HandleGameEnd(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
		return
	}

	err := GlobalGameService.EndGame(ctx, u.RoomID, u.ID)
	if err != nil {
		sendError(u, err.Error())
		return
	}
	// Success notification is handled by the Service through Broadcaster
}

// HandleGameAction 플레이어 행동
func HandleGameAction(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
		return
	}

	actionData, ok := event.Data["action"].(map[string]interface{})
	if !ok {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	err := GlobalGameService.ProcessAction(ctx, u.RoomID, u.ID, actionData)
	if err != nil {
		sendError(u, err.Error())
		return
	}
	// Success notification is handled by the Service through Broadcaster
}

// HandleGameSync 게임 상태 동기화 (클라이언트가 명시적으로 요청 시)
func HandleGameSync(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
		return
	}

	state, gameMode, err := GlobalGameService.GetGameState(ctx, u.RoomID, u.ID)
	if err != nil {
		sendError(u, err.Error())
		return
	}

	sendResult(u, event.Type, map[string]any{
		"roomId":    u.RoomID,
		"gameMode":  gameMode,
		"gameState": state,
	}, resp.SuccessCodeGameAction)
}

func HandleGamePause(ctx context.Context, user *user.Session, event SocketEvent) {
	sendError(user, resp.ErrorCodeGameFeatureNotImplemented)
}

// HandleGameInfo 현재 설정된 게임 모드 정보 (게임방법 조회)
func HandleGameInfo(ctx context.Context, user *user.Session, event SocketEvent) {
	gameModeStr, ok := event.Data["gameMode"].(string)
	if !ok {
		sendError(user, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	gameMode, info, err := GlobalGameService.GetGameInfo(gameModeStr)
	if err != nil {
		sendError(user, err.Error())
		return
	}

	sendResult(user, event.Type, map[string]any{
		"gameMode": gameMode,
		"info":     info,
	}, resp.SuccessCodeSystemOK)
}
