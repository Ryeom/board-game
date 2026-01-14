package ws

import (
	"context"
	"time"

	"github.com/Ryeom/board-game/internal/domain/room"
	"github.com/Ryeom/board-game/internal/game"
	"github.com/Ryeom/board-game/internal/game/hanabi"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
)

var activeGameEngines = make(map[string]game.Engine)

// HandleGameStart (game.start)게임 시작
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
		//if playerSession.Conn == nil || playerSession.Status != "ready" {
		if playerSession.Status != "ready" { // 임시
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
	case game.ModeHanabi:
		hanabiEngine := hanabi.NewEngine(
			playersInRoom,
			func(eventName string, playerIDs []string, state any) {
				fullHanabiState, ok := state.(*hanabi.State)
				if !ok {
					log.Logger.Errorf("HandleGameStart BroadcastFunc: Invalid state type, expected *hanabi.State")
					return
				}

				for _, pID := range playerIDs {
					playerView := fullHanabiState.GetPlayerView(pID)
					payload := map[string]any{
						"state": playerView,
					}
					res := createWebSocketResult(eventName, payload, resp.SuccessCodeGameSync, "ko")
					GlobalBroadcaster.SendToPlayer(pID, res) // start:1 (게임에 활용하기 위한 데이터) to All (game.started.init)
				}
			},
			setGameStateFunc,
			getGameStateFunc,
		)
		engine = hanabiEngine
	case game.Mode6Nimmt:
	case game.ModeTilePush:
		// (rows, cols 변수 정의는 그대로 유지)
		//tilePushEngine := tilepush.NewEngine(
		//	playersInRoom,
		//	func(eventName string, playerIDs []string, state any) {
		//		fullTilePushState, ok := state.(*tilepush.State)
		//		if !ok {
		//			log.Logger.Errorf("HandleGameStart BroadcastFunc: Invalid state type, expected *tilepush.State")
		//			return
		//		}
		//
		//		for _, pID := range playerIDs {
		//			playerView := fullTilePushState.GetPlayerView(pID)
		//			payload := GameStatePayload{
		//				RoomId:              r.ID,
		//				GameMode:            r.GameMode,
		//				GameStatus:          game.StatusPlaying,
		//				GameState:           playerView,
		//				CurrentTurnPlayerId: playerView.CurrentTurnPlayerID,
		//				Timestamp:           time.Now(),
		//			}
		//			res := createWebSocketResult(eventName, payload, resp.SuccessCodeGameSync, "ko")
		//			GlobalBroadcaster.SendToPlayer(pID, res)
		//		}
		//	},
		//	func(state *tilepush.State) error {
		//		return game.SaveGameState(ctx, r.GameMode, r.ID, state)
		//	},
		//	func() *tilepush.State {
		//		var loadedState tilepush.State
		//		err := game.GetGameState(ctx, r.GameMode, r.ID, &loadedState)
		//		if err != nil {
		//			log.Logger.Warningf("HandleGameStart - Could not load existing game state for room %s (mode %s): %v. Creating new.", r.ID, r.GameMode, err)
		//			return nil
		//		}
		//		return &loadedState
		//	},
		//)
		//engine = tilePushEngine
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
	payload := GameStatePayload{
		RoomId:     r.ID,
		GameMode:   r.GameMode,
		Timestamp:  time.Now(),
		GameStatus: game.StatusPlaying,
	}
	res := createWebSocketResult("game.started", payload, resp.SuccessCodeGameSync, "ko")
	// start:2 (host에 방생성 성공 알림) to host
	sendResult(u, event.Type, res, resp.SuccessCodeGameStart)
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
	payload := GameStatePayload{
		RoomId:     r.ID,
		GameMode:   r.GameMode,
		Timestamp:  time.Now(),
		GameStatus: game.StatusDefault,
	}
	res := createWebSocketResult("game.ended", payload, resp.SuccessCodeGameSync, "ko")
	GlobalBroadcaster.BroadcastToRoom(r.ID, res)

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
	case game.ModeHanabi:
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

	// action:1 (actor가 생성한 이벤트의 반영) to ALL : game.action.sync
	err := specificEngine.HandleEvent(gameEvent)
	if err != nil {
		log.Logger.Errorf("HandleGameAction - Game engine error for room %s, action %s: %v", u.RoomID, gameEvent.Type, err)
		sendError(u, resp.ErrorCodeGameActionFailed)
		return
	}

	// 게임이 종료되었는지 확인하고, 종료되었으면 `HandleGameEnd`를 호출
	if specificEngine.IsGameOver() {
		log.Logger.Infof("Game in room %s ended automatically. Calling HandleGameEnd.", r.ID)
		gameEndEvent := SocketEvent{
			Type: "game.end",
			Data: map[string]interface{}{"roomId": r.ID},
		}
		HandleGameEnd(ctx, u, gameEndEvent)
		return
	}

	payload := GameStatePayload{
		RoomId:     r.ID,
		GameMode:   r.GameMode,
		Timestamp:  time.Now(),
		GameStatus: game.StatusPlaying,
	}
	res := createWebSocketResult("game.action.succeeded", payload, resp.SuccessCodeGameAction, "ko")
	// action:2 (actor가 생성한 이벤트 성공 알림) to actor
	sendResult(u, event.Type, res, resp.SuccessCodeGameStart)
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
	case game.ModeHanabi:
		hanabiEng, typeOk := engine.(*hanabi.Engine)
		if !typeOk {
			log.Logger.Errorf("HandleGameSync - Mismatched engine type for room %s: expected hanabi.Engine", u.RoomID)
			sendError(u, resp.ErrorCodeGameSyncFailed)
			return
		}
		// 현재 요청한 플레이어의 시점에서 본 게임 상태를 생성
		currentGameState = hanabiEng.CurrentState.GetPlayerView(u.ID)
	default:
		sendError(u, resp.ErrorCodeRoomUnsupportedGameMode)
		return
	}

	sendResult(u, event.Type, map[string]any{
		"roomId":    u.RoomID,
		"gameMode":  r.GameMode,
		"gameState": currentGameState,
	}, resp.SuccessCodeGameAction)
}

func HandleGamePause(ctx context.Context, user *user.Session, event SocketEvent) {
	sendError(user, resp.ErrorCodeGameFeatureNotImplemented)
}

// HandleGameInfo 현재 설정된 게임 모드 정보 (게임방법 조회)
func HandleGameInfo(ctx context.Context, user *user.Session, event SocketEvent) {
	gameModeStr, ok := event.Data["gameMode"].(string)
	if !ok || gameModeStr == "" {
		sendError(user, resp.ErrorCodeRoomInvalidRequest)
		return
	}

	gameMode := game.Mode(gameModeStr)
	var gameInfo map[string]any

	switch gameMode {
	case game.ModeHanabi:
		gameInfo = map[string]any{
			"name":        "Hanabi",
			"description": "하나비는 협력 카드 게임입니다. 플레이어들은 불꽃놀이를 완성하기 위해 카드 정보를 공유하며 색깔별로 1부터 5까지 순서대로 카드를 내야 합니다. 하지만 자신의 패는 볼 수 없습니다!",
			"rulesSummary": []string{
				"각 플레이어는 4~5장의 카드를 받습니다 (인원수에 따라 다름).",
				"자신의 카드는 볼 수 없지만, 다른 플레이어의 카드는 볼 수 있습니다.",
				"턴에는 힌트 주기, 카드 내려놓기, 카드 버리기 중 하나를 수행합니다.",
				"힌트는 색상 또는 숫자에 대해 줄 수 있으며, 힌트 토큰을 소모합니다.",
				"카드를 내려놓을 때는 올바른 순서대로 내려놓아야 합니다. 실패하면 미스 토큰을 잃습니다.",
				"카드를 버리면 힌트 토큰을 얻습니다.",
				"미스 토큰 3개를 잃거나 모든 불꽃놀이를 완성하면 게임이 종료됩니다.",
				"덱이 소진되면 모든 플레이어가 마지막 턴을 진행한 후 게임이 종료됩니다.",
			},
			"cardDistribution": map[string]int{
				"1s": 3, "2s": 2, "3s": 2, "4s": 2, "5s": 1,
			},
			"initialTokens": map[string]int{
				"hint": 8, "miss": 3,
			},
		}
	default:
		sendError(user, resp.ErrorCodeRoomUnsupportedGameMode)
		return
	}

	sendResult(user, event.Type, map[string]any{
		"gameMode": gameMode,
		"info":     gameInfo,
	}, resp.SuccessCodeSystemOK)
}
