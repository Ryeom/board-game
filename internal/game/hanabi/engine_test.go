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

func TestEndGame_PerfectScore(t *testing.T) {
	players := []string{"p1", "p2"}
	var lastEvent string
	var lastPayload any
	engine := NewEngine(players,
		func(eventName string, playerIDs []string, state any) {
			lastEvent = eventName
			lastPayload = state
		},
		func(state *State) error { return nil },
		func() *State { return nil },
	)

	state := newTestState()
	state.Fireworks = map[Color]int{
		Red: 5, Green: 5, Blue: 5, Yellow: 5, White: 5,
	}
	state.MissTokens = 3
	engine.CurrentState = state

	engine.EndGame()

	if lastEvent != "game.end" {
		t.Errorf("이벤트가 game.end여야 하지만 %s", lastEvent)
	}
	endState, ok := lastPayload.(*State)
	if !ok {
		t.Fatal("브로드캐스트 페이로드가 *State 타입이 아님")
	}
	if endState.FinalScore != 25 {
		t.Errorf("FinalScore가 25여야 하지만 %d", endState.FinalScore)
	}
	if endState.EndReason != "perfect" {
		t.Errorf("EndReason이 perfect여야 하지만 %s", endState.EndReason)
	}
}

func TestEndGame_MissDepleted(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.Fireworks = map[Color]int{
		Red: 3, Green: 2, Blue: 1, Yellow: 0, White: 4,
	}
	state.MissTokens = 0
	state.GameOver = true
	engine.CurrentState = state

	engine.EndGame()

	if engine.CurrentState.FinalScore != 10 {
		t.Errorf("FinalScore가 10이어야 하지만 %d", engine.CurrentState.FinalScore)
	}
	if engine.CurrentState.EndReason != "miss_depleted" {
		t.Errorf("EndReason이 miss_depleted여야 하지만 %s", engine.CurrentState.EndReason)
	}
}

func TestEndGame_DeckExhausted(t *testing.T) {
	players := []string{"p1", "p2"}
	engine := newTestEngine(players)

	state := newTestState()
	state.Fireworks = map[Color]int{
		Red: 5, Green: 4, Blue: 3, Yellow: 2, White: 1,
	}
	state.MissTokens = 2
	state.GameOver = true
	engine.CurrentState = state

	engine.EndGame()

	if engine.CurrentState.FinalScore != 15 {
		t.Errorf("FinalScore가 15여야 하지만 %d", engine.CurrentState.FinalScore)
	}
	if engine.CurrentState.EndReason != "deck_exhausted" {
		t.Errorf("EndReason이 deck_exhausted여야 하지만 %s", engine.CurrentState.EndReason)
	}
}

// TestGameFlow_FullCycle 게임 시작부터 종료까지 전체 플로우 통합 테스트
func TestGameFlow_FullCycle(t *testing.T) {
	players := []string{"p1", "p2"}
	var events []string

	engine := NewEngine(players,
		func(eventName string, playerIDs []string, state any) {
			events = append(events, eventName)
		},
		func(state *State) error { return nil },
		func() *State { return nil },
	)

	// StartGame 호출 → 상태 초기화 확인
	engine.StartGame()

	if engine.CurrentState == nil {
		t.Fatal("StartGame 후 CurrentState가 nil")
	}
	if !engine.CurrentState.GameStarted {
		t.Error("GameStarted가 true여야 함")
	}
	if engine.CurrentState.HintTokens != 8 {
		t.Errorf("초기 힌트 토큰이 8이어야 하지만 %d", engine.CurrentState.HintTokens)
	}
	if engine.CurrentState.MissTokens != 3 {
		t.Errorf("초기 미스 토큰이 3이어야 하지만 %d", engine.CurrentState.MissTokens)
	}
	if len(engine.CurrentState.PlayerHands["p1"]) != 5 {
		t.Errorf("p1 초기 카드가 5장이어야 하지만 %d장", len(engine.CurrentState.PlayerHands["p1"]))
	}
	if len(engine.CurrentState.PlayerHands["p2"]) != 5 {
		t.Errorf("p2 초기 카드가 5장이어야 하지만 %d장", len(engine.CurrentState.PlayerHands["p2"]))
	}

	// game.start.init 브로드캐스트 확인
	if len(events) == 0 || events[0] != "game.start.init" {
		t.Errorf("첫 이벤트가 game.start.init이어야 하지만 %v", events)
	}

	// p1이 p2에게 힌트 (p2의 첫 번째 카드 색상)
	p2FirstCard := engine.CurrentState.PlayerHands["p2"][0]
	err := engine.HandleEvent(Event{
		Type: "give_hint",
		Data: map[string]any{
			"playerId": "p1",
			"toId":     "p2",
			"hintType": "color",
			"value":    string(p2FirstCard.Color),
		},
	})
	if err != nil {
		t.Fatalf("힌트 주기 실패: %v", err)
	}
	if engine.CurrentState.HintTokens != 7 {
		t.Errorf("힌트 후 토큰이 7이어야 하지만 %d", engine.CurrentState.HintTokens)
	}

	// 턴 종료 → p2 차례로 전환
	err = engine.HandleEvent(Event{Type: "end_turn", Data: map[string]any{}})
	if err != nil {
		t.Fatalf("턴 종료 실패: %v", err)
	}
	if engine.CurrentState.TurnIndex != 1 {
		t.Errorf("턴 종료 후 TurnIndex가 1이어야 하지만 %d", engine.CurrentState.TurnIndex)
	}

	// p2가 카드 버리기 → 힌트 토큰 회복
	err = engine.HandleEvent(Event{
		Type: "discard",
		Data: map[string]any{
			"playerId":  "p2",
			"cardIndex": float64(0),
		},
	})
	if err != nil {
		t.Fatalf("버리기 실패: %v", err)
	}
	if engine.CurrentState.HintTokens != 8 {
		t.Errorf("버리기 후 힌트 토큰이 8이어야 하지만 %d", engine.CurrentState.HintTokens)
	}
	if len(engine.CurrentState.DiscardPile) != 1 {
		t.Errorf("버린 카드 더미에 1장이어야 하지만 %d장", len(engine.CurrentState.DiscardPile))
	}

	// 게임이 아직 끝나지 않았는지 확인
	if engine.IsGameOver() {
		t.Error("아직 게임이 종료되면 안 됨")
	}
}

// TestGameFlow_MissTokenDepletion 미스 토큰 소진으로 게임 종료되는 전체 흐름
func TestGameFlow_MissTokenDepletion(t *testing.T) {
	players := []string{"p1", "p2"}
	var events []string

	engine := NewEngine(players,
		func(eventName string, playerIDs []string, state any) {
			events = append(events, eventName)
		},
		func(state *State) error { return nil },
		func() *State { return nil },
	)

	// 미스 토큰 1인 상태에서 시작 (빠른 종료 테스트용)
	// 제어된 덱으로 직접 상태 구성
	deck := []*Card{
		{Color: Red, Number: 1},
		{Color: Green, Number: 1},
	}
	state := NewState(deck)
	state.GameStarted = true
	state.TurnIndex = 0
	state.LastPlayer = -1
	state.MissTokens = 1 // 한 번만 틀리면 게임 종료
	state.Fireworks = map[Color]int{
		Red: 0, Green: 0, Blue: 0, Yellow: 0, White: 0,
	}
	// p1에게 일부러 틀린 카드 배치 (Blue 3을 내면 Red 위에 놓으려 하지만 실패)
	state.PlayerHands["p1"] = []*Card{
		{Color: Blue, Number: 3}, // Fireworks[Blue]=0이므로 3을 내면 실패
	}
	state.PlayerHands["p2"] = []*Card{
		{Color: Red, Number: 1},
	}
	engine.CurrentState = state

	// p1이 잘못된 카드를 냄 → 미스 토큰 소진 → 게임 종료
	err := engine.HandleEvent(Event{
		Type: "play_card",
		Data: map[string]any{
			"playerId":  "p1",
			"cardIndex": float64(0),
		},
	})
	if err != nil {
		t.Fatalf("카드 내기 실패: %v", err)
	}

	if engine.CurrentState.MissTokens != 0 {
		t.Errorf("미스 토큰이 0이어야 하지만 %d", engine.CurrentState.MissTokens)
	}
	if !engine.IsGameOver() {
		t.Error("미스 토큰 소진 후 게임이 종료되어야 함")
	}

	// EndGame 호출 → FinalScore, EndReason 확인
	engine.EndGame()

	if engine.CurrentState.EndReason != "miss_depleted" {
		t.Errorf("EndReason이 miss_depleted여야 하지만 %s", engine.CurrentState.EndReason)
	}
	if engine.CurrentState.FinalScore != 0 {
		t.Errorf("FinalScore가 0이어야 하지만 %d", engine.CurrentState.FinalScore)
	}

	// game.end 브로드캐스트 확인
	lastEvent := events[len(events)-1]
	if lastEvent != "game.end" {
		t.Errorf("마지막 이벤트가 game.end여야 하지만 %s", lastEvent)
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
