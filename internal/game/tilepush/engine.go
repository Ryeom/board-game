package tilepush

import (
	"fmt"

	"github.com/Ryeom/board-game/internal/domain/tilepush"
)

type Event struct {
	Type string
	Data map[string]any
}

type BroadcastFunc func(eventName string, playerIDs []string, state any)
type SetGameStateFunc func(state *State) error
type GetGameStateFunc func() *State

type Engine struct {
	Players      []string
	Broadcast    BroadcastFunc
	SetGameState SetGameStateFunc
	GetGameState GetGameStateFunc
	CurrentState *State
}

func NewEngine(players []string, broadcast BroadcastFunc, setGameState SetGameStateFunc, getGameState GetGameStateFunc) *Engine {
	return &Engine{
		Players:      players,
		Broadcast:    broadcast,
		SetGameState: setGameState,
		GetGameState: getGameState,
	}
}

func (e *Engine) StartGame() {
	fmt.Println("[TilePush] StartGame")
	state := e.GetGameState()
	if state == nil {
		tileSet, err := tilepush.GetRandomTileSet()
		if err != nil {
			fmt.Printf("[TilePush] Failed to get tile set: %v\n", err)
			return
		}

		state = NewState(e.Players, tileSet, 5, 5)
		e.CurrentState = state
	} else {
		fmt.Println("[TilePush] Resuming game with existing state.")
	}

	if err := e.SetGameState(e.CurrentState); err != nil {
		fmt.Printf("[TilePush] Error saving game state on start: %v\n", err)
	}

	e.Broadcast("game.start.init", e.Players, e.CurrentState)
}

func (e *Engine) EndGame() {
	fmt.Println("[TilePush] EndGame")
	e.Broadcast("game.end", e.Players, e.CurrentState)
}

func (e *Engine) HandleEvent(event any) error {
	cast, ok := event.(Event)
	if !ok {
		return fmt.Errorf("invalid event type")
	}
	fmt.Println("[TilePush] HandleEvent - Type:", cast.Type)

	var err error
	switch cast.Type {
	case "tile.push":
		err = e.handleTilePush(cast.Data)
	default:
		return fmt.Errorf("unknown event type: %s", cast.Type)
	}

	if err != nil {
		return err
	}

	if e.IsGameOver() {
		e.EndGame()
	}

	if saveErr := e.SetGameState(e.CurrentState); saveErr != nil {
		fmt.Printf("[TilePush] Error saving game state after event %s: %v\n", cast.Type, saveErr)
	}

	e.Broadcast("game.action.sync", e.Players, e.CurrentState)

	return nil
}

func (e *Engine) handleTilePush(data map[string]any) error {
	// TODO
	// - 플레이어의 턴인지 확인 (e.CurrentState.CurrentTurnPlayerID == 요청한 플레이어 ID)
	// - 덱에서 타일 하나 뽑기 (e.CurrentState.Deck)
	// - 요청된 보드 위치 (행/열) 및 타일 유효성 검증
	// - 보드에 타일 배치 및 밀려나온 타일 처리
	// - 밀려나온 타일 종류에 따라 턴을 유지할지, 상대방에게 넘길지 결정
	// - 게임 종료 조건 확인 (예: 덱 소진, 보드 가득 참 등)

	if len(e.CurrentState.Deck) == 0 {
		return fmt.Errorf("deck is empty, cannot draw tile")
	}
	drawnTile := e.CurrentState.Deck[0]
	e.CurrentState.Deck = e.CurrentState.Deck[1:]

	// TODO: 보드에 drawnTile을 배치하고, 밀려나온 타일을 처리하는 로직
	// TODO: 턴 전환 로직 (밀려나온 타일과 규칙에 따라)

	fmt.Printf("[TilePush] Player %s pushed tile %s\n", data["playerId"], drawnTile.Shape)
	return nil
}

func (e *Engine) IsGameOver() bool {
	if e.CurrentState == nil {
		return false
	}
	if len(e.CurrentState.Deck) == 0 {
		e.CurrentState.GameOver = true
	}
	return e.CurrentState.GameOver
}
