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
	Rows                int               `json:"rows"`
	Columns             int               `json:"columns"`
	PlayerHands         map[string][]Tile `json:"playerHands"`
	CurrentTurnPlayerID string            `json:"currentTurnPlayerId"`
	ActiveTileSet       *tilepush.TileSet `json:"activeTileSet"`
	GameOver            bool              `json:"gameOver"`
	Deck                []Tile            `json:"deck"`
	DiscardPile         []Tile            `json:"discardPile"`
	PlayerTargets       map[string]Tile   `json:"playerTargets"` // 각 플레이어의 목표 타일 (어떤 타일을 모으는지)
}

var seededRand *rand.Rand

func init() {
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func NewState(players []string, tileSet *tilepush.TileSet, rows, columns int) *State {
	board := make(Board, rows)
	for r := range board {
		board[r] = make([]Tile, columns)
		for c := range board[r] {
			board[r][c] = Tile{}
		}
	}

	var deck []Tile
	if tileSet != nil {
		const numCopiesPerTileType = 5
		for _, t := range tileSet.Tiles {
			for i := 0; i < numCopiesPerTileType; i++ {
				deck = append(deck, Tile{Shape: t.Shape, ImageURL: t.ImageURL})
			}
		}
	}
	shuffleTiles(deck)

	playerTargets := make(map[string]Tile)
	if len(players) >= 2 && tileSet != nil && len(tileSet.Tiles) >= 2 {
		playerTargets[players[0]] = tileSet.Tiles[0]
		playerTargets[players[1]] = tileSet.Tiles[1]
	}

	for c := 0; c < columns; c++ {
		if len(deck) > 0 {
			board[rows-1][c] = deck[0]
			deck = deck[1:]
		} else {
			break
		}
	}

	currentTurnPlayerID := ""
	if len(players) > 0 {
		currentTurnPlayerID = players[0]
	}

	return &State{
		Board:               board,
		Rows:                rows,
		Columns:             columns,
		PlayerHands:         make(map[string][]Tile),
		CurrentTurnPlayerID: currentTurnPlayerID,
		ActiveTileSet:       tileSet,
		GameOver:            false,
		Deck:                deck,
		DiscardPile:         []Tile{},
		PlayerTargets:       playerTargets,
	}
}

func shuffleTiles(tiles []Tile) {
	seededRand.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
}

func (s *State) IsGameOver() bool {
	return s.GameOver || len(s.Deck) == 0
}

func (s *State) GetPlayerView(playerID string) *State {
	playerView := *s
	return &playerView
}
