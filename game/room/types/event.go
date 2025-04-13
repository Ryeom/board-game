package types

type EventType string

const (
	EventJoin      EventType = "join"
	EventStartGame EventType = "start_game"
	EventPlayCard  EventType = "play_card"
	EventGiveHint  EventType = "give_hint"
	EventDiscard   EventType = "discard"
	EventEndTurn   EventType = "end_turn"
	// 필요한 만큼 확장 가능
)

type Event struct {
	Type EventType      `json:"type"`
	Data map[string]any `json:"data"` // 자유롭게 커스터마이징 가능한 필드
}
