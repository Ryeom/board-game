package hanabi

import (
	"testing"
)

func TestHandlePlayCard_VictoryCondition(t *testing.T) {
	// 1. 엔진 설정: Mock 브로드캐스트 및 콜백 함수
	mockBroadcast := func(eventName string, playerIDs []string, state any) {
	}
	mockSetState := func(state *State) error { return nil }
	mockGetState := func() *State { return nil }

	players := []string{"p1", "p2"}
	engine := NewEngine(players, mockBroadcast, mockSetState, mockGetState)

	// 2. 승리 직전 상태를 수동으로 구성
	// 승리 조건: 25점 (5가지 색상 각 5점)
	// 4색 완성 + White만 4까지 진행된 상태로 설정
	deck := GenerateDeck()
	state := NewState(deck)
	state.GameStarted = true
	state.Fireworks = map[Color]int{
		Red: 5, Green: 5, Blue: 5, Yellow: 5, White: 4,
	}
	// p1에게 마지막 카드(White 5) 부여
	state.PlayerHands["p1"] = []*Card{
		{Color: White, Number: 5, ColorKnown: true, NumberKnown: true},
	}
	state.PlayerHands["p2"] = []*Card{} // 더미

	engine.CurrentState = state

	// 3. 액션: p1이 승리 카드를 낸다 (인덱스 0)
	actionData := map[string]any{
		"playerId":  "p1",
		"cardIndex": float64(0),
	}

	// 4. handlePlayCard 실행
	err := engine.handlePlayCard(actionData)
	if err != nil {
		t.Fatalf("handlePlayCard 실패: %v", err)
	}

	// 5. 검증: 25점 도달 시 GameOver가 true여야 한다
	if !engine.CurrentState.GameOver {
		t.Errorf("25점 도달 후 GameOver가 true여야 하지만 false")
	}

	// 점수 검증
	if score := engine.CurrentState.GetCurrentScore(); score != 25 {
		t.Errorf("점수가 25여야 하지만 %d", score)
	}
}
