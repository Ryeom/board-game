package hanabi

import (
	"math/rand"
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

var seededRand *rand.Rand

func init() {
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
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

// DealInitialCards 게임 시작 시 플레이어에 초기 카드 분배
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
	seededRand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})
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
				// 힌트 정보가 없으면 카드 정보 삭제
				// 힌트를 받은 카드(ColorKnown 또는 NumberKnown이 true)는 정보를 유지
				if !copiedCard.ColorKnown {
					copiedCard.Color = ""
				}
				if !copiedCard.NumberKnown {
					copiedCard.Number = 0
				}
			}
			copiedHand[i] = &copiedCard
		}
		playerView.PlayerHands[pID] = copiedHand
	}
	return &playerView
}

// GetCardsRemainingInDeck 현재 덱에 남은 카드의 수 반환
func (s *State) GetCardsRemainingInDeck() int {
	return len(s.Deck)
}

// GetCurrentScore 불꽃놀이 완성 점수 (ending에 영향)
func (s *State) GetCurrentScore() int {
	score := 0
	for _, num := range s.Fireworks {
		score += num
	}
	return score
}

// GetRemainingHintTokens 현재 남은 힌트 수 반환
func (s *State) GetRemainingHintTokens() int {
	return s.HintTokens
}

// IsGameOver 게임이 종료 조건을 충족했는지 여부 반환
func (s *State) IsGameOver() bool {
	return s.GameOver
}

// GetTurnsRemaining 게임이 몇 턴 후에 끝나는지 반환
func (s *State) GetTurnsRemaining(numPlayers int) int {
	if s.GameOver {
		return 0
	}
	if len(s.Deck) > 0 { // deck 남아있는 경우
		return -1
	}

	if s.LastPlayer == -1 {
		return numPlayers
	}
	turnsLeft := (s.LastPlayer + numPlayers - s.TurnIndex) % numPlayers
	if turnsLeft == 0 && s.TurnIndex != s.LastPlayer {
		return numPlayers
	} else if turnsLeft == 0 && s.TurnIndex == s.LastPlayer {
		return 1
	}
	// current -> 라스트플레이어로 전환 = (라스트플레이어 - 턴인덱스 + numPlayers) % numPlayers
	// + 1 하기
	remaining := (s.LastPlayer - s.TurnIndex + numPlayers) % numPlayers
	if remaining == 0 && s.TurnIndex == s.LastPlayer {
		return 1
	}
	return remaining + 1
}
