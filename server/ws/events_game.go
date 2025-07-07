package ws

import (
	"context"

	"github.com/Ryeom/board-game/internal/domain/room"
	"github.com/Ryeom/board-game/internal/game"
	"github.com/Ryeom/board-game/internal/game/hanabi"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
)

var activeGameEngines = make(map[string]*hanabi.Engine)

func HandleGameStart(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
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

	if len(r.Players) < 2 { // 최소 플레이어 수 확인 (예시)
		sendError(u, resp.ErrorCodeGameNotEnoughPlayers)
		return
	}

	// 이미 게임이 시작되었는지 확인
	if _, exists := activeGameEngines[r.ID]; exists {
		sendError(u, resp.ErrorCodeGameAlreadyStarted)
		return
	}

	setGameStateFunc := func(state *hanabi.State) error {
		return game.SaveGameState(ctx, r.GameMode, r.ID, state)
	}

	getGameStateFunc := func() *hanabi.State {
		var loadedState hanabi.State

		err := game.GetGameState(ctx, r.GameMode, r.ID, &loadedState)
		if err != nil {
			log.Logger.Warningf("HandleGameStart - Could not load existing game state for room %s (mode %s): %v. Creating new.", r.ID, r.GameMode, err)
			return nil
		}
		return &loadedState
	}

	playersInRoom := r.Players

	engine := hanabi.NewEngine(
		playersInRoom,
		func(playerIDs []string, state any) {

			GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
				"type": "game.state",
				"data": state,
			})
		},
		setGameStateFunc,
		getGameStateFunc,
	)

	engine.StartGame()
	activeGameEngines[r.ID] = engine

	sendResult(u, event.Type, map[string]any{
		"roomId":    r.ID,
		"gameMode":  r.GameMode,
		"gameState": engine.CurrentState,
	}, resp.SuccessCodeGameStart)

	GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
		"type": "game.started",
		"data": map[string]string{
			"roomId": r.ID,
			"hostId": u.ID,
		},
	})
}

func HandleGameEnd(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
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

	delete(activeGameEngines, r.ID)

	if err := game.DeleteGameState(ctx, r.GameMode, r.ID); err != nil {
		log.Logger.Errorf("HandleGameEnd - Failed to delete game state for room %s (mode %s): %v", r.ID, r.GameMode, err)
		sendError(u, resp.ErrorCodeGameActionFailed)
		return
	}

	GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
		"type": "game.ended",
		"data": map[string]string{
			"roomId": r.ID,
			"hostId": u.ID,
		},
	})

	sendResult(u, event.Type, map[string]any{
		"roomId": r.ID,
		"status": "ended",
	}, resp.SuccessCodeGameEnd)
}

func HandleGameAction(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
		return
	}

	engine, ok := activeGameEngines[u.RoomID]
	if !ok {
		sendError(u, resp.ErrorCodeGameNotStarted)
		return
	}

	actionData, ok := event.Data["action"].(map[string]interface{})
	if !ok {
		sendError(u, resp.ErrorCodeRoomInvalidRequest) // TODO : 오류 변경
		return
	}

	actionData["playerId"] = u.ID

	hanabiEvent := hanabi.Event{
		Type: actionData["actionType"].(string),
		Data: actionData,
	}

	err := engine.HandleEvent(hanabiEvent)
	if err != nil {
		log.Logger.Errorf("HandleGameAction - Game engine error for room %s, action %s: %v", u.RoomID, hanabiEvent.Type, err)
		sendError(u, resp.ErrorCodeGameActionFailed)
		return
	}

	sendResult(u, event.Type, map[string]any{
		"status": "action processed",
		"action": hanabiEvent.Type,
	}, resp.SuccessCodeGameAction)
}

func HandleGameSync(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
		return
	}

	engine, ok := activeGameEngines[u.RoomID]
	if !ok || engine.CurrentState == nil {
		sendError(u, resp.ErrorCodeGameNotStarted)
		return
	}

	sendResult(u, event.Type, map[string]any{
		"roomId":    u.RoomID,
		"gameMode":  room.GameModeHanabi,
		"gameState": engine.CurrentState,
	}, resp.SuccessCodeGameSync)
}

func HandleGamePause(ctx context.Context, user *user.Session, event SocketEvent) {
	sendError(user, resp.ErrorCodeGameFeatureNotImplemented)
}

func HandleGameInfo(ctx context.Context, user *user.Session, event SocketEvent) {
	sendError(user, resp.ErrorCodeGameFeatureNotImplemented)
}
