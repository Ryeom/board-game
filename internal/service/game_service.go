package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Ryeom/board-game/internal/domain/room"
	"github.com/Ryeom/board-game/internal/game"
	"github.com/Ryeom/board-game/internal/game/hanabi"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/log"
)

type Broadcaster interface {
	SendToPlayer(playerID string, eventName string, payload any, msgCode string)
	BroadcastToRoom(roomID string, eventName string, payload any, msgCode string)
}

type GameService struct {
	Manager     *game.Manager
	Broadcaster Broadcaster
}

func NewGameService(manager *game.Manager, broadcaster Broadcaster) *GameService {
	return &GameService{
		Manager:     manager,
		Broadcaster: broadcaster,
	}
}

func (s *GameService) StartGame(ctx context.Context, roomID string, userID string) error {
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}

	if r.Host != userID {
		return fmt.Errorf(resp.ErrorCodeRoomNotHost)
	}

	if len(r.Players) < 2 {
		return fmt.Errorf(resp.ErrorCodeGameNotEnoughPlayers)
	}

	if r.IsGameStarted {
		return fmt.Errorf(resp.ErrorCodeGameAlreadyStarted)
	}

	if !r.AllPlayersReady() {
		return fmt.Errorf(resp.ErrorCodeGameNotAllPlayersReady)
	}

	setGameStateFunc := func(state *hanabi.State) error {
		return game.SaveGameState(ctx, r.GameMode, r.ID, state)
	}

	getGameStateFunc := func() *hanabi.State {
		var loadedState hanabi.State
		err := game.GetGameState(ctx, r.GameMode, r.ID, &loadedState)
		if err != nil {
			log.Logger.Warningf("StartGame - Could not load existing game state for room %s: %v. Creating new.", r.ID, err)
			return nil
		}
		return &loadedState
	}

	var engine game.Engine

	switch r.GameMode {
	case game.ModeHanabi:
		hanabiEngine := hanabi.NewEngine(
			r.Players,
			func(eventName string, playerIDs []string, state any) {
				fullHanabiState, ok := state.(*hanabi.State)
				if !ok {
					log.Logger.Errorf("BroadcastFunc: Invalid state type, expected *hanabi.State")
					return
				}
				for _, pID := range playerIDs {
					playerView := fullHanabiState.GetPlayerView(pID)
					payload := map[string]any{
						"state": playerView,
					}
					s.Broadcaster.SendToPlayer(pID, eventName, payload, resp.SuccessCodeGameSync)
				}
			},
			setGameStateFunc,
			getGameStateFunc,
		)
		engine = hanabiEngine
	default:
		return fmt.Errorf(resp.ErrorCodeRoomUnsupportedGameMode)
	}

	engine.StartGame()
	s.Manager.AddEngine(r.ID, engine)

	r.IsGameStarted = true
	r.ResetReady()
	if err := r.Save(); err != nil {
		return fmt.Errorf(resp.ErrorCodeGameInfoNotSaved)
	}

	// Payload construction for 'game.started'
	payload := map[string]any{
		"roomId":     r.ID,
		"gameMode":   r.GameMode,
		"timestamp":  time.Now(),
		"gameStatus": game.StatusPlaying,
	}
	// Notify the host specifically that game started (handled by caller or here?)
	// The original code passed 'start:2' to host specifically.
	// We can use Broadcaster to send uniqueness.
	s.Broadcaster.SendToPlayer(userID, "game.started", payload, resp.SuccessCodeGameStart)

	return nil
}

func (s *GameService) EndGame(ctx context.Context, roomID string, userID string) error {
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}
	if r.Host != userID {
		return fmt.Errorf(resp.ErrorCodeRoomNotHost)
	}

	s.Manager.RemoveEngine(r.ID)

	if err := game.DeleteGameState(ctx, r.GameMode, r.ID); err != nil {
		log.Logger.Errorf("EndGame - Failed to delete game state: %v", err)
		return fmt.Errorf(resp.ErrorCodeGameStateNotDeleted)
	}

	r.IsGameStarted = false
	r.ResetReady()
	if err := r.Save(); err != nil {
		return fmt.Errorf(resp.ErrorCodeGameInfoNotSaved)
	}

	payload := map[string]any{
		"roomId":     r.ID,
		"gameMode":   r.GameMode,
		"timestamp":  time.Now(),
		"gameStatus": game.StatusDefault,
	}
	s.Broadcaster.BroadcastToRoom(r.ID, "game.ended", payload, resp.SuccessCodeGameSync)
	return nil
}

func (s *GameService) ProcessAction(ctx context.Context, roomID string, userID string, actionData map[string]any) error {
	engine, ok := s.Manager.GetEngine(roomID)
	if !ok {
		return fmt.Errorf(resp.ErrorCodeGameNotStarted)
	}

	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}

	var specificEngine game.Engine
	switch r.GameMode {
	case game.ModeHanabi:
		hanabiEng, typeOk := engine.(*hanabi.Engine)
		if !typeOk {
			return fmt.Errorf(resp.ErrorCodeGameActionFailed)
		}
		specificEngine = hanabiEng
	default:
		return fmt.Errorf(resp.ErrorCodeRoomUnsupportedGameMode)
	}

	actionData["playerId"] = userID
	gameEvent := hanabi.Event{
		Type: actionData["actionType"].(string),
		Data: actionData,
	}

	if err := specificEngine.HandleEvent(gameEvent); err != nil {
		log.Logger.Errorf("ProcessAction - Engine error: %v", err)
		return fmt.Errorf(resp.ErrorCodeGameActionFailed)
	}

	if specificEngine.IsGameOver() {
		log.Logger.Infof("Game in room %s ended automatically.", roomID)

		// 엔진이 플레이어별 뷰로 game.end 브로드캐스트
		specificEngine.EndGame()

		// 정리: 엔진 제거, 게임 상태 삭제, 방 상태 업데이트
		s.Manager.RemoveEngine(r.ID)
		if err := game.DeleteGameState(ctx, r.GameMode, r.ID); err != nil {
			log.Logger.Errorf("ProcessAction - Failed to delete game state: %v", err)
		}
		r.IsGameStarted = false
		r.ResetReady()
		if err := r.Save(); err != nil {
			log.Logger.Errorf("ProcessAction - Failed to save room state: %v", err)
		}
		return nil
	}

	payload := map[string]any{
		"roomId":     r.ID,
		"gameMode":   r.GameMode,
		"timestamp":  time.Now(),
		"gameStatus": game.StatusPlaying,
	}
	s.Broadcaster.SendToPlayer(userID, "game.action.succeeded", payload, resp.SuccessCodeGameAction)
	return nil
}

func (s *GameService) GetGameState(ctx context.Context, roomID string, userID string) (any, game.Mode, error) {
	engine, ok := s.Manager.GetEngine(roomID)
	if !ok {
		return nil, "", fmt.Errorf(resp.ErrorCodeGameNotStarted)
	}
	r, ok := room.GetRoom(ctx, roomID)
	if !ok {
		return nil, "", fmt.Errorf(resp.ErrorCodeRoomNotFound)
	}

	switch r.GameMode {
	case game.ModeHanabi:
		hanabiEng, typeOk := engine.(*hanabi.Engine)
		if !typeOk {
			return nil, "", fmt.Errorf(resp.ErrorCodeGameSyncFailed)
		}
		state := hanabiEng.CurrentState.GetPlayerView(userID)
		return state, r.GameMode, nil
	default:
		return nil, "", fmt.Errorf(resp.ErrorCodeRoomUnsupportedGameMode)
	}
}

func (s *GameService) GetGameInfo(gameModeStr string) (game.Mode, map[string]any, error) {
	if gameModeStr == "" {
		return "", nil, fmt.Errorf(resp.ErrorCodeRoomInvalidRequest)
	}
	gameMode := game.Mode(gameModeStr)
	
	switch gameMode {
	case game.ModeHanabi:
		info := map[string]any{
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
		return gameMode, info, nil
	default:
		return "", nil, fmt.Errorf(resp.ErrorCodeRoomUnsupportedGameMode)
	}
}
