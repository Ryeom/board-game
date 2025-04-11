package hanabi

import "encoding/json"

type EventType string

const (
	EventJoin     EventType = "join"
	EventPlayCard           = "play_card"
	// ...
)

type Event struct {
	Type EventType       `json:"type"`
	Data json.RawMessage `json:"data"`
}
