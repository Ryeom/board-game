package hanabi

import (
	"fmt"
)

type Event struct {
	Type string
	Data map[string]any
}

type BroadcastFunc func(playerIDs []string, state any)

type Engine struct {
	Players       []string
	Broadcast     BroadcastFunc
	SetGameState  func(state any)
	GetPlayerFunc func() []string
	CurrentState  *State
}

func NewEngine(players []string, broadcast BroadcastFunc, setState func(any), getPlayers func() []string) *Engine {
	return &Engine{
		Players:       players,
		Broadcast:     broadcast,
		SetGameState:  setState,
		GetPlayerFunc: getPlayers,
	}
}

func (e *Engine) StartGame() {
	fmt.Println("[Hanabi] StartGame")

	deck := GenerateDeck()
	state := NewState(deck)
	DealInitialCards(e.Players, &state.Deck, state.PlayerHands)

	state.GameStarted = true
	state.TurnIndex = 0
	state.LastPlayer = (len(e.Players) + state.TurnIndex - 1) % len(e.Players)

	e.CurrentState = state
	e.SetGameState(state)
	e.Broadcast(e.Players, state)
}

func (e *Engine) HandleEvent(event any) error {
	cast, ok := event.(Event)
	if !ok {
		return fmt.Errorf("invalid event")
	}
	fmt.Println("[Hanabi] HandleEvent - Type:", cast.Type)
	switch cast.Type {
	case "give_hint":
		return e.handleGiveHint(cast.Data)
	case "play_card":
		return e.handlePlayCard(cast.Data)
	case "discard":
		return e.handleDiscardCard(cast.Data)
	case "end_turn":
		return e.handleEndTurn()
	default:
		return fmt.Errorf("unknown event type: %s", cast.Type)
	}
}

func (e *Engine) handleGiveHint(data map[string]any) error {
	//fromID, _ := data["fromId"].(string)
	toID, _ := data["toId"].(string)
	hintType, _ := data["hintType"].(string)
	value := data["value"]

	hand := e.CurrentState.PlayerHands[toID]
	switch hintType {
	case "color":
		colorStr, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid color hint")
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
			return fmt.Errorf("invalid number hint")
		}
		for _, card := range hand {
			if card.Number == int(num) {
				card.NumberKnown = true
			}
		}
	default:
		return fmt.Errorf("unknown hint type")
	}

	if e.CurrentState.HintTokens > 0 {
		e.CurrentState.HintTokens--
	}
	e.Broadcast(e.Players, e.CurrentState)
	return nil
}

func (e *Engine) handlePlayCard(data map[string]any) error {
	playerID, _ := data["playerId"].(string)
	index, _ := data["cardIndex"].(float64)

	hand := e.CurrentState.PlayerHands[playerID]
	if int(index) >= len(hand) {
		return fmt.Errorf("invalid card index")
	}

	card := hand[int(index)]
	e.CurrentState.PlayerHands[playerID] = append(hand[:int(index)], hand[int(index)+1:]...)

	if e.CurrentState.Fireworks[card.Color]+1 == card.Number {
		e.CurrentState.Fireworks[card.Color]++
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
	e.Broadcast(e.Players, e.CurrentState)
	return nil
}

func (e *Engine) handleDiscardCard(data map[string]any) error {
	playerID, _ := data["playerId"].(string)
	index, _ := data["cardIndex"].(float64)

	hand := e.CurrentState.PlayerHands[playerID]
	if int(index) >= len(hand) {
		return fmt.Errorf("invalid card index")
	}

	card := hand[int(index)]
	e.CurrentState.DiscardPile = append(e.CurrentState.DiscardPile, card)
	e.CurrentState.PlayerHands[playerID] = append(hand[:int(index)], hand[int(index)+1:]...)
	if e.CurrentState.HintTokens < 8 {
		e.CurrentState.HintTokens++
	}
	e.Broadcast(e.Players, e.CurrentState)
	return nil
}

func (e *Engine) handleEndTurn() error {
	e.CurrentState.TurnIndex = (e.CurrentState.TurnIndex + 1) % len(e.Players)
	if len(e.CurrentState.Deck) == 0 && e.CurrentState.TurnIndex == (e.CurrentState.LastPlayer+1)%len(e.Players) {
		e.CurrentState.GameOver = true
	}
	e.Broadcast(e.Players, e.CurrentState)
	return nil
}
