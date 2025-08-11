package tilepush

import (
	"math/rand"
	"time"

	"github.com/Ryeom/board-game/internal/domain/tilepush"
)

type Tile = tilepush.Tile

type Board [][]Tile

type State struct {
	Board               Board             `json:"board"`
	Rows                int               `json:"rows"`                // 보드의 행 수
	Columns             int               `json:"columns"`             // 보드의 열 수
	PlayerHands         map[string][]Tile `json:"playerHands"`         // 플레이어 손패 (타일 푸시 게임에서는 사용하지 않을 수 있습니다)
	CurrentTurnPlayerID string            `json:"currentTurnPlayerId"` // 현재 턴 플레이어 ID
	ActiveTileSet       *tilepush.TileSet `json:"activeTileSet"`       // 현재 게임에 사용 중인 타일 세트 정보
	GameOver            bool              `json:"gameOver"`
	Deck                []Tile            `json:"deck"`        // 게임에 사용될 남은 타일 덱
	DiscardPile         []Tile            `json:"discardPile"` // 버려진 타일 더미 (게임 종료 조건 등에 활용 가능)
}

var seededRand *rand.Rand

func init() {
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func NewState(players []string, tileSet *tilepush.TileSet, rows, columns int) *State {
	// max row 8, colums 4~5
	// 1. 게임 보드 초기화
	board := make(Board, rows)
	for r := range board {
		board[r] = make([]Tile, columns)
		for c := range board[r] {
			board[r][c] = Tile{}
		}
	}

	// 2. 덱 생성 및 "반만 섞기" 로직 적용
	var deck []Tile
	if tileSet != nil {
		const numCopiesPerTileType = 5
		for _, t := range tileSet.Tiles {
			for i := 0; i < numCopiesPerTileType; i++ {
				deck = append(deck, Tile{Shape: t.Shape, ImageURL: t.ImageURL})
			}
		}
	}

	deckLen := len(deck)
	if deckLen > 1 {
		halfPoint := deckLen / 2

		seededRand.Shuffle(halfPoint, func(i, j int) {
			deck[i], deck[j] = deck[j], deck[i]
		})

		seededRand.Shuffle(deckLen-halfPoint, func(i, j int) {
			deck[halfPoint+i], deck[halfPoint+j] = deck[halfPoint+j], deck[halfPoint+i]
		})
	}

	// 3. 플레이어 손패 초기화 (타일 푸시 규칙에 따라 손패가 없을 수도 있습니다)
	playerHands := make(map[string][]Tile)

	// 4. 초기 턴 플레이어 설정
	currentTurnPlayerID := ""
	if len(players) > 0 {
		currentTurnPlayerID = players[0]
	}

	return &State{
		Board:               board,
		Rows:                rows,
		Columns:             columns,
		PlayerHands:         playerHands,
		CurrentTurnPlayerID: currentTurnPlayerID,
		ActiveTileSet:       tileSet,
		GameOver:            false,
		Deck:                deck,
		DiscardPile:         []Tile{},
	}
}

func (s *State) IsGameOver() bool {
	return s.GameOver || len(s.Deck) == 0 // 덱이 비면 게임 종료로 간주 (기본)
}

func (s *State) GetPlayerView(playerID string) *State {
	playerView := *s
	return &playerView
}
