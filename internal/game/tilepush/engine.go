package tilepush

import (
	"errors"
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

		state = NewState(e.Players, tileSet, 5, 5) // 예시: 5x5 보드
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
	// 1. 현재 턴 플레이어인지 확인
	playerID, ok := data["playerId"].(string)
	if !ok || playerID == "" {
		return errors.New("invalid player ID")
	}
	if e.CurrentState.CurrentTurnPlayerID != playerID {
		return fmt.Errorf("not %s's turn", playerID)
	}

	// 2. 덱에서 타일 하나 뽑기
	if len(e.CurrentState.Deck) == 0 {
		e.CurrentState.GameOver = true
		return errors.New("deck is empty, game over")
	}
	drawnTile := e.CurrentState.Deck[0]
	e.CurrentState.Deck = e.CurrentState.Deck[1:]

	colFloat, colOk := data["column"].(float64)
	if !colOk {
		return errors.New("invalid column index")
	}
	col := int(colFloat)

	if col < 0 || col >= e.CurrentState.Columns {
		return errors.New("column index out of board bounds")
	}

	// 4. 타일 삽입 위치 (row) 결정 및 유효성 검증
	insertRowFloat, insertRowOk := data["row"].(float64)
	if !insertRowOk {
		return errors.New("missing insert row index")
	}
	insertRow := int(insertRowFloat)

	if insertRow < 0 || insertRow >= e.CurrentState.Rows {
		return errors.New("insert row index out of board bounds")
	}
	// 5. 보드에 타일 배치 및 밀려나온 타일 처리
	var pushedOutTile Tile
	if e.CurrentState.Board[e.CurrentState.Rows-1][col].Shape != "" {
		pushedOutTile = e.CurrentState.Board[e.CurrentState.Rows-1][col]
	}

	for rIdx := e.CurrentState.Rows - 1; rIdx > insertRow; rIdx-- {
		e.CurrentState.Board[rIdx][col] = e.CurrentState.Board[rIdx-1][col]
	}
	e.CurrentState.Board[insertRow][col] = drawnTile

	if pushedOutTile.Shape != "" {
		e.CurrentState.DiscardPile = append(e.CurrentState.DiscardPile, pushedOutTile)
	}

	// 6. 턴 전환 로직: 밀어 넣은 타일의 종류와 밀려나온 타일의 종류를 비교
	if drawnTile.Shape == pushedOutTile.Shape {
		fmt.Printf("[TilePush] Player %s pushed a matching tile (%s). Turn stays with %s.\n", playerID, drawnTile.Shape, playerID)
	} else {
		// 턴 넘기기
		currentIndex := -1
		for i, p := range e.Players {
			if p == playerID {
				currentIndex = i
				break
			}
		}
		if currentIndex != -1 {
			e.CurrentState.CurrentTurnPlayerID = e.Players[(currentIndex+1)%len(e.Players)]
			fmt.Printf("[TilePush] Player %s pushed a non-matching tile (%s vs %s). Turn passes to %s.\n", playerID, drawnTile.Shape, pushedOutTile.Shape, e.CurrentState.CurrentTurnPlayerID)
		} else {
			return errors.New("current player not found in player list")
		}
	}

	fmt.Printf("[TilePush] Player %s pushed tile %s into (%d, %d). Pushed out: %s\n", playerID, drawnTile.Shape, insertRow, col, pushedOutTile.Shape)
	return nil
}

func (e *Engine) IsGameOver() bool {
	if e.CurrentState == nil {
		return false
	}
	if len(e.CurrentState.Deck) == 0 {
		e.CurrentState.GameOver = true // 덱이 비면 게임 종료로 설정
	}
	return e.CurrentState.GameOver
}
