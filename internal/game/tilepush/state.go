package tilepush

import (
	"github.com/Ryeom/board-game/internal/domain/tilepush"
	"math/rand"
	"time"
)

type Tile = tilepush.Tile

type Board [][]Tile

type State struct {
	Board               Board             `json:"board"`
	Rows                int               `json:"rows"`
	Columns             int               `json:"columns"`
	PlayerHands         map[string][]Tile `json:"playerHands"` // 플레이어 손패 (게임 규칙에 따라 사용 여부 결정)
	CurrentTurnPlayerID string            `json:"currentTurnPlayerId"`
	ActiveTileSet       *tilepush.TileSet `json:"activeTileSet"` // 현재 게임에 사용 중인 타일 세트 정보
	GameOver            bool              `json:"gameOver"`
	Deck                []Tile            `json:"deck"` // 게임에 사용될 남은 타일 덱
}

var seededRand *rand.Rand

func init() {
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func NewState(players []string, tileSet *tilepush.TileSet, rows, columns int) *State {
	// max row 8, colums 4~5
	// 1. 게임 보드 초기화
	board := make(Board, rows)
	for i := range board {
		board[i] = make([]Tile, columns)
	}

	// 2. 덱 생성
	var deck []Tile
	if tileSet != nil {
		for _, t := range tileSet.Tiles {
			deck = append(deck, Tile{Shape: t.Shape, ImageURL: t.ImageURL})
		}
	}
	shuffleTiles(deck)

	// 3. 플레이어 손패 초기화
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
	}
}

func shuffleTiles(tiles []Tile) {
	seededRand.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
}

func (s *State) IsGameOver() bool {
	return s.GameOver
}

func (s *State) GetPlayerView(playerID string) *State {
	playerView := *s
	return &playerView
}
