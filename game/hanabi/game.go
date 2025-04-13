package hanabi

type Game struct {
	TurnIndex   int
	Deck        []*Card
	DiscardPile []*Card
	Fireworks   map[Color]int
	HintTokens  int
	MissTokens  int
	// 기타 로직 함수들 (PlayCard, GiveHint 등)
}
