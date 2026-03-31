package hanabi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStrategyTestState() *State {
	return &State{
		Fireworks: map[Color]int{
			Red: 0, Green: 0, Blue: 0, Yellow: 0, White: 0,
		},
		HintTokens:  MaxHintTokens,
		MissTokens:  InitialMissTokens,
		PlayerHands: make(map[string][]*Card),
	}
}

func TestHeuristicStrategy_SafePlay(t *testing.T) {
	s := newStrategyTestState()
	s.Fireworks[Red] = 2
	s.PlayerHands["ai_1"] = []*Card{
		{Color: Blue, Number: 3, ColorKnown: false, NumberKnown: false},
		{Color: Red, Number: 3, ColorKnown: true, NumberKnown: true},
	}
	s.PlayerHands["human"] = []*Card{
		{Color: Green, Number: 1},
	}

	strategy := &HeuristicStrategy{}
	action, data, err := strategy.Decide([]string{"ai_1", "human"}, "ai_1", s)

	require.NoError(t, err)
	assert.Equal(t, "play_card", action)
	assert.Equal(t, float64(1), data["cardIndex"])
}

func TestHeuristicStrategy_SafePlay_NotKnown(t *testing.T) {
	// ColorKnown만 true이고 NumberKnown이 false면 안전하지 않음
	s := newStrategyTestState()
	s.Fireworks[Red] = 2
	s.HintTokens = 0
	s.PlayerHands["ai_1"] = []*Card{
		{Color: Red, Number: 3, ColorKnown: true, NumberKnown: false},
	}
	s.PlayerHands["human"] = []*Card{
		{Color: Green, Number: 1},
	}

	strategy := &HeuristicStrategy{}
	action, _, err := strategy.Decide([]string{"ai_1", "human"}, "ai_1", s)

	require.NoError(t, err)
	// 힌트도 못 주고 (토큰 0), 버리기도 못 하고 (토큰 만석 아님 → 가능)
	// HintTokens=0 < MaxHintTokens이므로 discard 가능
	assert.Equal(t, "discard", action)
}

func TestHeuristicStrategy_UsefulHint(t *testing.T) {
	s := newStrategyTestState()
	s.Fireworks[Green] = 0
	s.HintTokens = 3
	s.PlayerHands["ai_1"] = []*Card{
		{Color: Blue, Number: 3, ColorKnown: false, NumberKnown: false},
	}
	s.PlayerHands["human"] = []*Card{
		{Color: Green, Number: 1, ColorKnown: false, NumberKnown: false},
	}

	strategy := &HeuristicStrategy{}
	action, data, err := strategy.Decide([]string{"ai_1", "human"}, "ai_1", s)

	require.NoError(t, err)
	assert.Equal(t, "give_hint", action)
	assert.Equal(t, "human", data["toId"])
	assert.Equal(t, "color", data["hintType"])
	assert.Equal(t, "green", data["value"])
}

func TestHeuristicStrategy_UsefulHint_NumberFallback(t *testing.T) {
	// 색은 이미 알고 있으면 숫자 힌트
	s := newStrategyTestState()
	s.Fireworks[Red] = 1
	s.HintTokens = 5
	s.PlayerHands["ai_1"] = []*Card{
		{Color: Blue, Number: 4, ColorKnown: false, NumberKnown: false},
	}
	s.PlayerHands["human"] = []*Card{
		{Color: Red, Number: 2, ColorKnown: true, NumberKnown: false},
	}

	strategy := &HeuristicStrategy{}
	action, data, err := strategy.Decide([]string{"ai_1", "human"}, "ai_1", s)

	require.NoError(t, err)
	assert.Equal(t, "give_hint", action)
	assert.Equal(t, "number", data["hintType"])
	assert.Equal(t, float64(2), data["value"])
}

func TestHeuristicStrategy_Discard(t *testing.T) {
	s := newStrategyTestState()
	s.HintTokens = 5 // 만석 아님
	s.PlayerHands["ai_1"] = []*Card{
		{Color: Red, Number: 1, ColorKnown: true, NumberKnown: false},
		{Color: Blue, Number: 3, ColorKnown: false, NumberKnown: false}, // 힌트 없음 → 이것부터 버림
	}
	s.PlayerHands["human"] = []*Card{
		{Color: Yellow, Number: 5, ColorKnown: false, NumberKnown: false}, // 플레이 불가 (Fireworks[Yellow]=0, 5≠1)
	}

	strategy := &HeuristicStrategy{}
	action, data, err := strategy.Decide([]string{"ai_1", "human"}, "ai_1", s)

	require.NoError(t, err)
	assert.Equal(t, "discard", action)
	assert.Equal(t, float64(1), data["cardIndex"]) // 힌트 없는 두 번째 카드
}

func TestHeuristicStrategy_ForcedPlay(t *testing.T) {
	// 힌트 토큰 만석 + 안전한 플레이 없음 + 유용한 힌트 없음 → 강제 플레이
	s := newStrategyTestState()
	s.HintTokens = MaxHintTokens
	s.PlayerHands["ai_1"] = []*Card{
		{Color: Red, Number: 5, ColorKnown: false, NumberKnown: false},
	}
	s.PlayerHands["human"] = []*Card{
		{Color: Yellow, Number: 5, ColorKnown: false, NumberKnown: false}, // 플레이 불가
	}

	strategy := &HeuristicStrategy{}
	action, data, err := strategy.Decide([]string{"ai_1", "human"}, "ai_1", s)

	require.NoError(t, err)
	assert.Equal(t, "play_card", action)
	assert.Equal(t, float64(0), data["cardIndex"])
}

func TestHeuristicStrategy_NoCards(t *testing.T) {
	s := newStrategyTestState()
	s.PlayerHands["ai_1"] = []*Card{}

	strategy := &HeuristicStrategy{}
	_, _, err := strategy.Decide([]string{"ai_1"}, "ai_1", s)

	assert.Error(t, err)
}
