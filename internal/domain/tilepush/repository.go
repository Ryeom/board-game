package tilepush

import (
	"context"
	"fmt"
	"github.com/Ryeom/board-game/infra/db"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

var (
	allTileSets []*TileSet
	seededRand  *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func LoadAllTileSetsFromDB(ctx context.Context) error {
	var tileSets []*TileSet
	result := db.DB.WithContext(ctx).Preload("Tiles").Find(&tileSets)
	if result.Error != nil {
		return fmt.Errorf("failed to load tile sets from DB: %w", result.Error)
	}
	allTileSets = tileSets
	return nil
}

func GetTileSetByName(ctx context.Context, name string) (*TileSet, error) {
	for _, ts := range allTileSets {
		if ts.Name == name {
			return ts, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func GetRandomTileSet() (*TileSet, error) {
	if len(allTileSets) == 0 {
		return nil, fmt.Errorf("no tile sets loaded from DB")
	}
	return allTileSets[seededRand.Intn(len(allTileSets))], nil
}
