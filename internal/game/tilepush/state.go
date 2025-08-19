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
	PlayerTargets       map[string]Tile   `json:"playerTargets"`      // 각 플레이어의 목표 타일 (어떤 타일을 모으는지)
	WinnerID            string            `json:"winnerId,omitempty"` // 승리한 플레이어 ID (게임 종료 시 설정)

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
	// 1. 덱이 비었을 때 게임 종료
	if len(s.Deck) == 0 {
		s.GameOver = true
		return true
	}

	// 2. 플레이어 승리 조건 확인
	for playerID, targetTile := range s.PlayerTargets {
		for c := 0; c < s.Columns; c++ {
			isColumnFullOfTarget := true
			for r := 0; r < s.Rows; r++ {
				if s.Board[r][c].Shape != targetTile.Shape {
					isColumnFullOfTarget = false
					break
				}
			}
			if isColumnFullOfTarget {
				s.GameOver = true
				s.WinnerID = playerID
				return true
			}
		}
		for r := 0; r < s.Rows; r++ {
			isRowFullOfTarget := true
			for c := 0; c < s.Columns; c++ {
				if s.Board[r][c].Shape != targetTile.Shape {
					isRowFullOfTarget = false
					break
				}
			}
			if isRowFullOfTarget {
				s.GameOver = true
				s.WinnerID = playerID
				return true
			}
		}
	}

	// 3. 보드판이 가득 찼을 때 (더 이상 놓을 곳이 없을 때) 게임 종료
	isBoardFull := true
	for r := 0; r < s.Rows; r++ {
		for c := 0; c < s.Columns; c++ {
			if s.Board[r][c].Shape == "" {
				isBoardFull = false
				break
			}
		}
		if !isBoardFull {
			break
		}
	}
	if isBoardFull {
		s.GameOver = true
		return true
	}

	return s.GameOver
}

func (s *State) GetPlayerView(playerID string) *State {
	playerView := *s
	return &playerView
}
