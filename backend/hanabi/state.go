package hanabi

type State struct {
	Fireworks   map[Color]int // 색상별로 쌓인 숫자 (1~5)
	HintTokens  int           // 남은 힌트 토큰
	MissTokens  int           // 남은 실수 토큰
	TurnIndex   int           // 현재 턴 주인공 (players[TurnIndex])
	Deck        []*Card       // 남은 카드
	DiscardPile []*Card       // 버려진 카드들
	GameStarted bool          // 게임 시작 여부
	GameOver    bool          // 게임 종료 여부
	LastPlayer  int           // 마지막 플레이어 인덱스 (덱 다 쓰면 한 바퀴 더 돌기용)
}

func NewState(deck []*Card) *State {
	return &State{
		Fireworks: map[Color]int{
			Red:    0,
			Green:  0,
			Blue:   0,
			White:  0,
			Yellow: 0,
		},
		HintTokens:  8,
		MissTokens:  3,
		TurnIndex:   0,
		Deck:        deck,
		DiscardPile: []*Card{},
		GameStarted: false,
		GameOver:    false,
	}
}
