package hanabi

import "math/rand"

/*	카드의 속성(색, 숫자), 초기 덱 구성 등 */
type Color string

const (
	Red    Color = "red"
	Green  Color = "green"
	Blue   Color = "blue"
	White  Color = "white"
	Yellow Color = "yellow"
)

type Card struct {
	Color  Color `json:"color"`
	Number int   `json:"number"`
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
		for num, count := range cardCounts {
			for i := 0; i < count; i++ {
				deck = append(deck, &Card{
					Color:  color,
					Number: num,
				})
			}
		}
	}

	shuffle(deck)
	return deck
}

func shuffle(cards []*Card) {
	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})
}

func DealInitialCards(players []*Attender, deck *[]*Card) {
	cardCount := 5
	if len(players) >= 4 {
		cardCount = 4
	}

	for _, player := range players {
		for i := 0; i < cardCount; i++ {
			if len(*deck) == 0 {
				return // 덱 다 썼으면 종료
			}
			player.Hand = append(player.Hand, (*deck)[0])
			*deck = (*deck)[1:]
		}
	}
}

type Hint struct {
	ColorKnown  *Color `json:"colorKnown,omitempty"`  // 색상이 알려졌을 경우
	NumberKnown *int   `json:"numberKnown,omitempty"` // 숫자가 알려졌을 경우
}
