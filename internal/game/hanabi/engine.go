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
	//e.Broadcast("", e.Players, state)
}

func (e *Engine) EndGame() {

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

	if e.CurrentState != nil {
		if saveErr := e.SetGameState(e.CurrentState); saveErr != nil {
			fmt.Printf("[Hanabi] Error saving game state after event %s: %v\n", cast.Type, saveErr)
		}
	}
	// e.Broadcast("",e.Players, e.CurrentState)
	return nil
}

func (e *Engine) handleGiveHint(data map[string]any) error {
	toID, _ := data["toId"].(string)
	hintType, _ := data["hintType"].(string)
	value := data["value"]

	if e.CurrentState == nil {
		return fmt.Errorf("game not started or state is nil")
	}

	hand, ok := e.CurrentState.PlayerHands[toID]
	if !ok {
		return fmt.Errorf("player %s hand not found", toID)
	}

	if e.CurrentState.HintTokens <= 0 {
		return fmt.Errorf("not enough hint tokens")
	}

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
			}
		}
	default:
		return fmt.Errorf("unknown hint type: %s", hintType)
	}

	e.CurrentState.HintTokens--
	return nil
}

func (e *Engine) handlePlayCard(data map[string]any) error {
	playerID, _ := data["playerId"].(string)
	indexFloat, _ := data["cardIndex"].(float64)
	index := int(indexFloat)

	if e.CurrentState == nil {
		return fmt.Errorf("game not started or state is nil")
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

	if e.CurrentState == nil {
		return fmt.Errorf("game not started or state is nil")
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
