package hanabi

import (
	"fmt"
	"game/room"
	"game/room/types"
)

// 🔥 하나비 게임 엔진 구조체
type Engine struct{}

// 🧱 인터페이스 구현체 생성
func NewEngine() *Engine {
	return &Engine{}
}

// ✅ 게임 시작 시 실행 (덱 생성 + 분배 + 상태 초기화)
func (e *Engine) StartGame(r *room.Room) {
	fmt.Println("[Hanabi] 게임 시작: RoomID =", r.ID)

	deck := GenerateDeck()
	state := NewState(deck)

	room.DealInitialCards(r.Players, &state.Deck)

	state.GameStarted = true
	state.TurnIndex = 0
	state.LastPlayer = (len(r.Players) + state.TurnIndex - 1) % len(r.Players)

	r.State = state
}

// ✅ WebSocket 메시지 처리
func (e *Engine) HandleEvent(r *room.Room, event types.Event) error {
	fmt.Println("[Hanabi] 이벤트 처리:", event.Type)

	switch event.Type {
	case types.EventGiveHint:
		// 힌트 처리 로직 (예: color or number)
		return e.handleGiveHint(r, event.Data)
	case types.EventPlayCard:
		// 카드 플레이 처리
	case types.EventDiscard:
		// 카드 버리기 처리
	case types.EventEndTurn:
		// 턴 넘기기
	default:
		return fmt.Errorf("알 수 없는 이벤트 타입: %s", event.Type)
	}

	return nil
}
