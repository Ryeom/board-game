package hanabi

import "fmt"

// Strategy AI 플레이어의 행동 결정 인터페이스
type Strategy interface {
	Decide(players []string, aiPlayerID string, state *State) (actionType string, actionData map[string]any, err error)
}

// HeuristicStrategy 규칙 기반 휴리스틱 전략
// 우선순위: 안전한 플레이 → 유용한 힌트 → 오래된 카드 버리기 → 강제 플레이
type HeuristicStrategy struct{}

func (h *HeuristicStrategy) Decide(players []string, aiPlayerID string, state *State) (string, map[string]any, error) {
	hand := state.PlayerHands[aiPlayerID]
	if len(hand) == 0 {
		return "", nil, fmt.Errorf("no cards in hand")
	}

	// 1. 안전한 플레이: ColorKnown + NumberKnown이고 Fireworks에 맞는 카드
	if action, data := h.trySafePlay(aiPlayerID, hand, state); action != "" {
		return action, data, nil
	}

	// 2. 유용한 힌트: 다른 플레이어의 플레이 가능한 카드에 힌트
	if state.HintTokens > 0 {
		if action, data := h.tryUsefulHint(players, aiPlayerID, state); action != "" {
			return action, data, nil
		}
	}

	// 3. 오래된 카드 버리기: 힌트 없는 카드 우선 (토큰 만석 아닐 때)
	if state.HintTokens < MaxHintTokens {
		if action, data := h.tryDiscard(aiPlayerID, hand); action != "" {
			return action, data, nil
		}
	}

	// 4. 강제 플레이: 위 조건 모두 불가 시 가장 오래된 카드 플레이
	return "play_card", map[string]any{
		"playerId":  aiPlayerID,
		"cardIndex": float64(0),
	}, nil
}

// trySafePlay 확실히 플레이 가능한 카드를 찾는다.
// AI는 ColorKnown, NumberKnown이 모두 true인 카드만 "아는" 것으로 간주 (치팅 방지).
func (h *HeuristicStrategy) trySafePlay(aiPlayerID string, hand []*Card, state *State) (string, map[string]any) {
	for i, card := range hand {
		if card.ColorKnown && card.NumberKnown {
			if state.Fireworks[card.Color]+1 == card.Number {
				return "play_card", map[string]any{
					"playerId":  aiPlayerID,
					"cardIndex": float64(i),
				}
			}
		}
	}
	return "", nil
}

// tryUsefulHint 다른 플레이어의 플레이 가능한 카드에 대해 힌트를 준다.
// 다른 플레이어의 카드는 실제로 볼 수 있으므로 (하나비 규칙) 전체 State를 참조한다.
func (h *HeuristicStrategy) tryUsefulHint(players []string, aiPlayerID string, state *State) (string, map[string]any) {
	for _, playerID := range players {
		if playerID == aiPlayerID {
			continue
		}
		otherHand := state.PlayerHands[playerID]
		for _, card := range otherHand {
			if state.Fireworks[card.Color]+1 != card.Number {
				continue
			}
			// 플레이 가능한 카드 발견 — 모르는 정보를 힌트로 제공
			if !card.ColorKnown {
				return "give_hint", map[string]any{
					"playerId": aiPlayerID,
					"toId":     playerID,
					"hintType": "color",
					"value":    string(card.Color),
				}
			}
			if !card.NumberKnown {
				return "give_hint", map[string]any{
					"playerId": aiPlayerID,
					"toId":     playerID,
					"hintType": "number",
					"value":    float64(card.Number),
				}
			}
			// 둘 다 이미 알고 있으면 다음 카드로
		}
	}
	return "", nil
}

// tryDiscard 힌트 정보가 없는 카드를 우선 버린다.
func (h *HeuristicStrategy) tryDiscard(aiPlayerID string, hand []*Card) (string, map[string]any) {
	// 힌트가 전혀 없는 카드 우선
	for i, card := range hand {
		if !card.ColorKnown && !card.NumberKnown {
			return "discard", map[string]any{
				"playerId":  aiPlayerID,
				"cardIndex": float64(i),
			}
		}
	}
	// 모든 카드에 힌트가 있으면 가장 오래된 카드 버리기
	return "discard", map[string]any{
		"playerId":  aiPlayerID,
		"cardIndex": float64(0),
	}
}
