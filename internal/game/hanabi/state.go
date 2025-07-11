package hanabi

import (
	"time"
)

type State struct {
	Fireworks   map[Color]int      `json:"fireworks"`
	HintTokens  int                `json:"hintTokens"`
	MissTokens  int                `json:"missTokens"`
	TurnIndex   int                `json:"turnIndex"`
	Deck        []*Card            `json:"-"`
	DiscardPile []*Card            `json:"discardPile"`
	GameStarted bool               `json:"gameStarted"`
	GameOver    bool               `json:"gameOver"`
	LastPlayer  int                `json:"lastPlayer"`
	PlayerHands map[string][]*Card `json:"playerHands"` // player ID → cards
}

func NewState(deck []*Card) *State {
	return &State{
		Fireworks: map[Color]int{
			Red: 0, Green: 0, Blue: 0, Yellow: 0, White: 0,
		},
		HintTokens:  8,
		MissTokens:  3,
		TurnIndex:   0,
		Deck:        deck,
		DiscardPile: []*Card{},
		GameStarted: false,
		GameOver:    false,
		PlayerHands: make(map[string][]*Card),
	}
}

func DealInitialCards(players []string, deck *[]*Card, hands map[string][]*Card) {
	cardCount := 5
	if len(players) >= 4 {
		cardCount = 4
	}

	for _, player := range players {
		hand := make([]*Card, 0, cardCount)
		for i := 0; i < cardCount; i++ {
			if len(*deck) == 0 {
				break
			}
			original := (*deck)[0]
			copy := &Card{
				Color:       original.Color,
				Number:      original.Number,
				ColorKnown:  false,
				NumberKnown: false,
			}
			hand = append(hand, copy)
			*deck = (*deck)[1:]
		}
		hands[player] = hand
	}
}

func GenerateDeck() []*Card {
	cardCounts := map[int]int{
		1: 3,
		2: 2,
		3: 2,
		4: 2,
		5: 1,
	}
	colors := []Color{Red, Green, Blue, Yellow, White}
	var deck []*Card
	for _, color := range colors {
		for number, count := range cardCounts {
			for i := 0; i < count; i++ {
				deck = append(deck, &Card{
					Color:  color,
					Number: number,
				})
			}
		}
	}
	shuffle(deck)
	return deck
}

func shuffle(cards []*Card) {
	for i := len(cards) - 1; i > 0; i-- {
		j := randInt(0, i+1)
		cards[i], cards[j] = cards[j], cards[i]
	}
}

func randInt(min, max int) int {
	return min + int(time.Now().UnixNano())%(max-min)
}

// GetPlayerView 특정 플레이어의 시점에서 본 게임 상태를 반환 (자신의 카드는 보이지 않아야함)
func (s *State) GetPlayerView(playerID string) *State {
	playerView := *s

	playerView.PlayerHands = make(map[string][]*Card)
	for pID, hand := range s.PlayerHands {
		copiedHand := make([]*Card, len(hand))
		for i, card := range hand {
			copiedCard := *card
			if pID == playerID {
				copiedCard.Color = ""
				copiedCard.Number = 0
				// ColorKnown, NumberKnown : 힌트가 있으면 유지하기
			}
			copiedHand[i] = &copiedCard
		}
		playerView.PlayerHands[pID] = copiedHand
	}
	return &playerView
}
