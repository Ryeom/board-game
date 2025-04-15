package types

type EventType string

const (
	EventJoin      EventType = "join"
	EventStartGame EventType = "start_game"
	EventPlayCard  EventType = "play_card"
	EventGiveHint  EventType = "give_hint"
	EventDiscard   EventType = "discard"
	EventEndTurn   EventType = "end_turn"
)

type Event struct {
	Type EventType      `json:"type"`
	Data map[string]any `json:"data"`
}
