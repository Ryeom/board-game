package hanabi

import (
	"fmt"
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
func (e *Engine) IsGameOver() bool {
	if e.CurrentState == nil {
		return false
	}
	return e.CurrentState.IsGameOver()
}

func (e *Engine) StartGame() {
	fmt.Println("[Hanabi] StartGame")

	state := e.GetGameState()
	if state == nil {
		deck := GenerateDeck()
		state = NewState(deck)
		state.PlayerHands = make(map[string][]*Card)
		DealInitialCards(e.Players, &state.Deck, state.PlayerHands)
		state.GameStarted = true
		state.TurnIndex = 0
		state.LastPlayer = -1
	} else {
		fmt.Println("[Hanabi] Resuming game with existing state.")
	}

	e.CurrentState = state
	if err := e.SetGameState(state); err != nil {
		fmt.Printf("[Hanabi] Error saving game state on start: %v\n", err)
	}
	e.Broadcast("game.start.init", e.Players, e.CurrentState)
}

func (e *Engine) EndGame() {

	e.Broadcast("game.end", e.Players, e.CurrentState)
}
func (e *Engine) HandleEvent(event any) error {
	cast, ok := event.(Event)
	if !ok {
		return fmt.Errorf("invalid event")
	}
	fmt.Println("[Hanabi] HandleEvent - Type:", cast.Type)
	var err error
	switch cast.Type {
	case "give_hint":
		err = e.handleGiveHint(cast.Data)
	case "play_card":
		err = e.handlePlayCard(cast.Data)
	case "discard":
		err = e.handleDiscardCard(cast.Data)
	case "end_turn":
		err = e.handleEndTurn()
	default:
		return fmt.Errorf("unknown event type: %s", cast.Type)
	}

	if err != nil {
		return err
	}

	// 추가된 로직: 게임 종료 조건을 확인합니다.
	if e.CurrentState.IsGameOver() {
		e.EndGame()
	}

	if e.CurrentState != nil {
		if saveErr := e.SetGameState(e.CurrentState); saveErr != nil {
			fmt.Printf("[Hanabi] Error saving game state after event %s: %v\n", cast.Type, saveErr)
		}
	}
	e.Broadcast("game.action.sync", e.Players, e.CurrentState)
	return nil
}

// currentPlayerID 현재 턴인 플레이어의 ID를 반환
func (e *Engine) currentPlayerID() string {
	return e.Players[e.CurrentState.TurnIndex]
}

// validateTurn 현재 턴인 플레이어인지 검증
func (e *Engine) validateTurn(playerID string) error {
	if e.CurrentState == nil {
		return fmt.Errorf("game not started or state is nil")
	}
	if e.currentPlayerID() != playerID {
		return fmt.Errorf("not your turn: current turn is %s", e.currentPlayerID())
	}
	return nil
}

func (e *Engine) handleGiveHint(data map[string]any) error {
	playerID, _ := data["playerId"].(string)
	toID, _ := data["toId"].(string)
	hintType, _ := data["hintType"].(string)
	value := data["value"]

	if err := e.validateTurn(playerID); err != nil {
		return err
	}

	// 자기 자신에게 힌트를 줄 수 없음
	if playerID == toID {
		return fmt.Errorf("cannot give hint to yourself")
	}

	hand, ok := e.CurrentState.PlayerHands[toID]
	if !ok {
		return fmt.Errorf("player %s hand not found", toID)
	}

	// 힌트 토큰이 0이면 힌트를 줄 수 없음
	if e.CurrentState.HintTokens <= 0 {
		return fmt.Errorf("no hint tokens remaining")
	}

	// 매칭되는 카드 수를 세서 빈 힌트 방지
	matchCount := 0
	switch hintType {
	case "color":
		colorStr, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid color hint value")
		}
		color := Color(colorStr)
		for _, card := range hand {
			if card.Color == color {
				card.ColorKnown = true
				matchCount++
			}
		}
	case "number":
		num, ok := value.(float64)
		if !ok {
			return fmt.Errorf("invalid number hint value")
		}
		for _, card := range hand {
			if card.Number == int(num) {
				card.NumberKnown = true
				matchCount++
			}
		}
	default:
		return fmt.Errorf("unknown hint type: %s", hintType)
	}

	if matchCount == 0 {
		return fmt.Errorf("hint must match at least one card")
	}

	e.CurrentState.HintTokens--
	return nil
}

func (e *Engine) handlePlayCard(data map[string]any) error {
	playerID, _ := data["playerId"].(string)
	indexFloat, _ := data["cardIndex"].(float64)
	index := int(indexFloat)

	if err := e.validateTurn(playerID); err != nil {
		return err
	}

	hand, ok := e.CurrentState.PlayerHands[playerID]
	if !ok || index < 0 || index >= len(hand) {
		return fmt.Errorf("invalid card index or player hand not found")
	}

	card := hand[index]

	e.CurrentState.PlayerHands[playerID] = append(hand[:index], hand[index+1:]...)

	if len(e.CurrentState.Deck) > 0 {
		newCardOriginal := e.CurrentState.Deck[0]
		newCardCopy := &Card{
			Color:       newCardOriginal.Color,
			Number:      newCardOriginal.Number,
			ColorKnown:  false,
			NumberKnown: false,
		}
		e.CurrentState.PlayerHands[playerID] = append(e.CurrentState.PlayerHands[playerID], newCardCopy)
		e.CurrentState.Deck = e.CurrentState.Deck[1:]
	} else {
		if e.CurrentState.LastPlayer == -1 {
			for i, pID := range e.Players {
				if pID == playerID {
					e.CurrentState.LastPlayer = i
					break
				}
			}
		}
	}

	if e.CurrentState.Fireworks[card.Color]+1 == card.Number {
		e.CurrentState.Fireworks[card.Color] = card.Number
		if card.Number == 5 && e.CurrentState.HintTokens < 8 {
			e.CurrentState.HintTokens++
		}
		// Victory Check: 승리 조건 25점, game over
		if e.CurrentState.GetCurrentScore() == 25 {
			e.CurrentState.GameOver = true
		}
	} else {
		e.CurrentState.DiscardPile = append(e.CurrentState.DiscardPile, card)
		e.CurrentState.MissTokens--
		if e.CurrentState.MissTokens <= 0 {
			e.CurrentState.GameOver = true
		}
	}
	return nil
}

func (e *Engine) handleDiscardCard(data map[string]any) error {
	playerID, _ := data["playerId"].(string)
	indexFloat, _ := data["cardIndex"].(float64)
	index := int(indexFloat)

	if err := e.validateTurn(playerID); err != nil {
		return err
	}

	// 힌트 토큰이 최대(8)일 때 버리기 불가
	if e.CurrentState.HintTokens >= 8 {
		return fmt.Errorf("cannot discard when hint tokens are full")
	}

	hand, ok := e.CurrentState.PlayerHands[playerID]
	if !ok || index < 0 || index >= len(hand) {
		return fmt.Errorf("invalid card index or player hand not found")
	}

	card := hand[index]
	e.CurrentState.DiscardPile = append(e.CurrentState.DiscardPile, card)

	e.CurrentState.PlayerHands[playerID] = append(hand[:index], hand[index+1:]...)

	if len(e.CurrentState.Deck) > 0 {
		newCardOriginal := e.CurrentState.Deck[0]
		newCardCopy := &Card{
			Color:       newCardOriginal.Color,
			Number:      newCardOriginal.Number,
			ColorKnown:  false,
			NumberKnown: false,
		}
		e.CurrentState.PlayerHands[playerID] = append(e.CurrentState.PlayerHands[playerID], newCardCopy)
		e.CurrentState.Deck = e.CurrentState.Deck[1:]
	} else {
		if e.CurrentState.LastPlayer == -1 {
			for i, pID := range e.Players {
				if pID == playerID {
					e.CurrentState.LastPlayer = i
					break
				}
			}
		}
	}

	if e.CurrentState.HintTokens < 8 {
		e.CurrentState.HintTokens++
	}
	return nil
}

func (e *Engine) handleEndTurn() error {
	if e.CurrentState == nil {
		return fmt.Errorf("game not started or state is nil")
	}

	e.CurrentState.TurnIndex = (e.CurrentState.TurnIndex + 1) % len(e.Players)

	if len(e.CurrentState.Deck) == 0 && e.CurrentState.LastPlayer != -1 && e.CurrentState.TurnIndex == (e.CurrentState.LastPlayer+1)%len(e.Players) {
		e.CurrentState.GameOver = true
	}
	return nil
}
