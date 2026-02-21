package hanabi

import (
	"strings"
	"testing"
)

// newTestEngine 테스트용 엔진 생성 헬퍼
func newTestEngine(players []string) *Engine {
	mockBroadcast := func(eventName string, playerIDs []string, state any) {}
	mockSetState := func(state *State) error { return nil }
	mockGetState := func() *State { return nil }
	return NewEngine(players, mockBroadcast, mockSetState, mockGetState)
}

// newTestState 기본 테스트 상태 생성 헬퍼
func newTestState() *State {
	state := NewState([]*Card{})
	state.GameStarted = true
	state.TurnIndex = 0
	state.LastPlayer = -1
	return state
}

func TestHandlePlayCard_VictoryCondition(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	// 승리 직전 상태: 4색 완성 + White 4까지 진행
	deck := GenerateDeck()
	state := NewState(deck)
	state.GameStarted = true
	state.Fireworks = map[Color]int{
		Red: 5, Green: 5, Blue: 5, Yellow: 5, White: 4,
	}
	state.PlayerHands["p1"] = []*Card{
		{Color: White, Number: 5, ColorKnown: true, NumberKnown: true},
	}
	state.PlayerHands["p2"] = []*Card{}

	engine.CurrentState = state

	// p1이 승리 카드를 낸다
	err := engine.handlePlayCard(map[string]any{
		"playerId":  "p1",
		"cardIndex": float64(0),
	})
	if err != nil {
		t.Fatalf("handlePlayCard 실패: %v", err)
	}

	if !engine.CurrentState.GameOver {
		t.Errorf("25점 도달 후 GameOver가 true여야 하지만 false")
	}
	if score := engine.CurrentState.GetCurrentScore(); score != 25 {
		t.Errorf("점수가 25여야 하지만 %d", score)
	}
}

func TestValidateTurn_WrongPlayer(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.TurnIndex = 0 // p1 차례
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// p2가 카드를 내려고 시도 → 실패해야 함
	err := engine.handlePlayCard(map[string]any{
		"playerId":  "p2",
		"cardIndex": float64(0),
	})
	if err == nil {
		t.Fatal("다른 플레이어 턴에 행동했는데 에러가 없음")
	}
	if !strings.Contains(err.Error(), "not your turn") {
		t.Errorf("턴 검증 에러 메시지가 예상과 다름: %v", err)
	}
}

func TestValidateTurn_CorrectPlayer(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.TurnIndex = 0 // p1 차례
	state.Fireworks[Red] = 0
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// p1이 카드를 냄 → 성공해야 함
	err := engine.handlePlayCard(map[string]any{
		"playerId":  "p1",
		"cardIndex": float64(0),
	})
	if err != nil {
		t.Fatalf("본인 턴에 행동했는데 에러 발생: %v", err)
	}
}

func TestGiveHint_ToSelf(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.HintTokens = 8
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// p1이 자기 자신에게 힌트 → 실패해야 함
	err := engine.handleGiveHint(map[string]any{
		"playerId": "p1",
		"toId":     "p1",
		"hintType": "color",
		"value":    "red",
	})
	if err == nil {
		t.Fatal("자기 자신에게 힌트를 줬는데 에러가 없음")
	}
	if !strings.Contains(err.Error(), "cannot give hint to yourself") {
		t.Errorf("에러 메시지가 예상과 다름: %v", err)
	}
}

func TestGiveHint_EmptyHint(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.HintTokens = 8
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// p1이 p2에게 "red" 힌트 → p2에 red 카드 없으므로 실패해야 함
	err := engine.handleGiveHint(map[string]any{
		"playerId": "p1",
		"toId":     "p2",
		"hintType": "color",
		"value":    "red",
	})
	if err == nil {
		t.Fatal("매칭 카드가 없는 힌트인데 에러가 없음")
	}
	if !strings.Contains(err.Error(), "hint must match at least one card") {
		t.Errorf("에러 메시지가 예상과 다름: %v", err)
	}

	// 힌트 토큰이 소모되지 않았는지 확인
	if state.HintTokens != 8 {
		t.Errorf("빈 힌트에 토큰이 소모됨: %d", state.HintTokens)
	}
}

func TestGiveHint_NoTokens(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.HintTokens = 0
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// 힌트 토큰 0인 상태에서 힌트 시도 → 실패해야 함
	err := engine.handleGiveHint(map[string]any{
		"playerId": "p1",
		"toId":     "p2",
		"hintType": "color",
		"value":    "blue",
	})
	if err == nil {
		t.Fatal("힌트 토큰이 0인데 에러가 없음")
	}
	if !strings.Contains(err.Error(), "no hint tokens remaining") {
		t.Errorf("에러 메시지가 예상과 다름: %v", err)
	}
}

func TestGiveHint_WrongTurn(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.TurnIndex = 0 // p1 차례
	state.HintTokens = 8
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// p2가 힌트 시도 → p1 차례이므로 실패해야 함
	err := engine.handleGiveHint(map[string]any{
		"playerId": "p2",
		"toId":     "p1",
		"hintType": "color",
		"value":    "red",
	})
	if err == nil {
		t.Fatal("다른 플레이어 턴에 힌트를 줬는데 에러가 없음")
	}
	if !strings.Contains(err.Error(), "not your turn") {
		t.Errorf("에러 메시지가 예상과 다름: %v", err)
	}
}

func TestDiscard_WhenHintTokensFull(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.HintTokens = 8 // 최대
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// 힌트 토큰 8(최대)인 상태에서 버리기 시도 → 실패해야 함
	err := engine.handleDiscardCard(map[string]any{
		"playerId":  "p1",
		"cardIndex": float64(0),
	})
	if err == nil {
		t.Fatal("힌트 토큰이 최대인데 버리기가 성공함")
	}
	if !strings.Contains(err.Error(), "cannot discard when hint tokens are full") {
		t.Errorf("에러 메시지가 예상과 다름: %v", err)
	}
}

func TestDiscard_WhenHintTokensNotFull(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.HintTokens = 7
	state.PlayerHands["p1"] = []*Card{{Color: Red, Number: 1}}
	state.PlayerHands["p2"] = []*Card{{Color: Blue, Number: 2}}
	engine.CurrentState = state

	// 힌트 토큰 7인 상태에서 버리기 → 성공, 토큰 8로 증가
	err := engine.handleDiscardCard(map[string]any{
		"playerId":  "p1",
		"cardIndex": float64(0),
	})
	if err != nil {
		t.Fatalf("버리기 실패: %v", err)
	}
	if state.HintTokens != 8 {
		t.Errorf("버리기 후 힌트 토큰이 8이어야 하지만 %d", state.HintTokens)
	}
}
