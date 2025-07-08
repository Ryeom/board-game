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

var activeGameEngines = make(map[string]game.Engine)

// HandleGameStart 게임 시작
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

	if r.IsGameStarted { // 방에 이미 게임이 시작된 경우
		sendError(u, resp.ErrorCodeGameAlreadyStarted)
		return
	}

	playersInRoomSessions, err := user.GetSessionsByRoom(r.ID)
	if err != nil {
		log.Logger.Errorf("HandleGameStart - Failed to get player sessions for room %s: %v", r.ID, err)
		sendError(u, resp.ErrorCodeGameAllUserNotReady)
		return
	}

	if len(playersInRoomSessions) != len(r.Players) {
		log.Logger.Warningf("HandleGameStart - Session count mismatch for room %s: %d active sessions vs %d players in room", r.ID, len(playersInRoomSessions), len(r.Players))
		// TODO : 세션이 없는 플레이어는 Room.Players 목록에서 제거
		sendError(u, resp.ErrorCodeGamePlayerNotInRoom)
		return
	}

	allPlayersReady := true
	for _, playerSession := range playersInRoomSessions {
		if playerSession.Conn == nil || playerSession.Status != "ready" {
			allPlayersReady = false
			log.Logger.Debugf("Player %s is not ready (status: %s, conn: %v)", playerSession.ID, playerSession.Status, playerSession.Conn != nil)
			break
		}
	}

	if !allPlayersReady {
		sendError(u, resp.ErrorCodeGameNotAllPlayersReady)
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

	var engine game.Engine

	switch r.GameMode {
	case room.GameModeHanabi:
		hanabiEngine := hanabi.NewEngine(
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
		engine = hanabiEngine
	default:
		sendError(u, resp.ErrorCodeRoomUnsupportedGameMode)
		return
	}

	engine.StartGame()
	activeGameEngines[r.ID] = engine

	r.IsGameStarted = true
	if err := r.Save(); err != nil {
		log.Logger.Errorf("HandleGameStart - Failed to save room state after game start for room %s: %v", r.ID, err)
		sendError(u, resp.ErrorCodeGameInfoNotSaved)
		return
	}

	var currentGameState interface{}
	if hanabiEng, ok := engine.(*hanabi.Engine); ok {
		currentGameState = hanabiEng.CurrentState
	}

	sendResult(u, event.Type, map[string]any{
		"roomId":    r.ID,
		"gameMode":  r.GameMode,
		"gameState": currentGameState, // 단언된 상태 전달
	}, resp.SuccessCodeGameStart)

	GlobalBroadcaster.BroadcastToRoom(r.ID, map[string]any{
		"type": "game.started",
		"data": map[string]string{
			"roomId": r.ID,
			"hostId": u.ID,
		},
	})
}

// HandleGameEnd 게임 종료
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
		sendError(u, resp.ErrorCodeGameStateNotDeleted)
		return
	}

	r.IsGameStarted = false
	if err := r.Save(); err != nil {
		log.Logger.Errorf("HandleGameEnd - Failed to save room state after game end for room %s: %v", r.ID, err)
		sendError(u, resp.ErrorCodeGameInfoNotSaved)
		return
	}

	playersInRoomSessions, err := user.GetSessionsByRoom(r.ID)
	if err != nil {
		log.Logger.Errorf("HandleGameEnd - Failed to get player sessions for room %s to reset status: %v", r.ID, err)
	}
	for _, playerSession := range playersInRoomSessions {
		playerSession.Status = "connected" // OR "idle", "waiting"
		if saveErr := user.SaveUserSession(playerSession); saveErr != nil {
			log.Logger.Errorf("HandleGameEnd - Failed to reset status for player %s: %v", playerSession.ID, saveErr)
		}
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

// HandleGameAction 플레이어 행동
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

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, resp.ErrorCodeRoomNotFound)
		return
	}

	var specificEngine game.Engine
	switch r.GameMode {
	case room.GameModeHanabi:
		hanabiEng, typeOk := engine.(*hanabi.Engine)
		if !typeOk {
			log.Logger.Errorf("HandleGameAction - Mismatched engine type for room %s: expected hanabi.Engine", u.RoomID)
			sendError(u, resp.ErrorCodeGameActionFailed)
			return
		}
		specificEngine = hanabiEng
	default:
		sendError(u, resp.ErrorCodeRoomUnsupportedGameMode)
		return
	}

	actionData, ok := event.Data["action"].(map[string]interface{})
	if !ok {
		sendError(u, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	actionData["playerId"] = u.ID

	// TODO : game에 따라 다른 데이터 반영
	gameEvent := hanabi.Event{
		Type: actionData["actionType"].(string),
		Data: actionData,
	}

	err := specificEngine.HandleEvent(gameEvent)
	if err != nil {
		log.Logger.Errorf("HandleGameAction - Game engine error for room %s, action %s: %v", u.RoomID, gameEvent.Type, err)
		sendError(u, resp.ErrorCodeGameActionFailed)
		return
	}

	sendResult(u, event.Type, map[string]any{
		"status": "action processed",
		"action": gameEvent.Type,
	}, resp.SuccessCodeGameAction)
}

// HandleGameSync 게임 상태 동기화 (클라이언트가 명시적으로 요청 시)
func HandleGameSync(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, resp.ErrorCodeChatNotInRoom)
		return
	}

	engine, ok := activeGameEngines[u.RoomID]
	if !ok {
		sendError(u, resp.ErrorCodeGameNotStarted)
		return
	}

	r, ok := room.GetRoom(ctx, u.RoomID)
	if !ok {
		sendError(u, resp.ErrorCodeRoomNotFound)
		return
	}

	var currentGameState interface{}
	switch r.GameMode {
	case room.GameModeHanabi:
		hanabiEng, typeOk := engine.(*hanabi.Engine)
		if !typeOk {
			log.Logger.Errorf("HandleGameSync - Mismatched engine type for room %s: expected hanabi.Engine", u.RoomID)
			sendError(u, resp.ErrorCodeGameSyncFailed)
			return
		}
		currentGameState = hanabiEng.CurrentState
	default:
		sendError(u, resp.ErrorCodeRoomUnsupportedGameMode)
		return
	}

	sendResult(u, event.Type, map[string]any{
		"roomId":    u.RoomID,
		"gameMode":  r.GameMode,
		"gameState": currentGameState,
	}, resp.SuccessCodeGameSync)
}

func HandleGamePause(ctx context.Context, user *user.Session, event SocketEvent) {
	sendError(user, resp.ErrorCodeGameFeatureNotImplemented)
}

func HandleGameInfo(ctx context.Context, user *user.Session, event SocketEvent) {
	sendError(user, resp.ErrorCodeGameFeatureNotImplemented)
}
