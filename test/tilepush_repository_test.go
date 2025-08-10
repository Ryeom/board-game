package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Ryeom/board-game/infra/db"
	"github.com/Ryeom/board-game/internal/domain/tilepush"
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/server"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// setupTestDB는 테스트를 위한 DB 연결 및 설정을 초기화합니다.
func setupTestDB() {
	oldArgs := os.Args
	os.Args = []string{oldArgs[0], "local"}
	defer func() { os.Args = oldArgs }()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current file path for test setup")
	}
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Dir(testDir)

	viper.AddConfigPath(projectRoot)
	viper.SetConfigName("settings")
	viper.SetConfigType("toml")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	err := l.InitializeApplicationLog()
	if err != nil {
		panic("Failed to initialize application log for tests: %v")
	}

	err = server.SetEnv()
	if err != nil {
		panic("Failed to set environment for tests: %v")
	}

	err = server.SetConfig()
	if err != nil {
		panic("Failed to load server configuration for tests: %v")
	}
	db.Initialize()

	if err := db.DB.AutoMigrate(&tilepush.TileSet{}, &tilepush.Tile{}); err != nil {
		panic(fmt.Errorf("failed to auto migrate tables: %w", err))
	}
}

func TestMain(m *testing.M) {
	setupTestDB()
	code := m.Run()

	os.Exit(code)
}

func TestLoadAllTileSetsFromDB(t *testing.T) {

	ctx := context.Background()
	err := tilepush.LoadAllTileSetsFromDB(ctx)

	assert.NoError(t, err)
}

func TestGetTileSetByName(t *testing.T) {
	ctx := context.Background()
	err := tilepush.LoadAllTileSetsFromDB(ctx)

	if err != nil {
		panic(err)
	}

	t.Run("Existing name", func(t *testing.T) {
		ts, err := tilepush.GetTileSetByName(ctx, "animals")
		assert.NoError(t, err)
		assert.NotNil(t, ts)
		assert.Equal(t, "animals", ts.Name)
	})

	t.Run("Non-existing name", func(t *testing.T) {
		ts, err := tilepush.GetTileSetByName(ctx, "insects")
		assert.Error(t, err)
		assert.Nil(t, ts)
	})
}

func TestGetRandomTileSet(t *testing.T) {
	ctx := context.Background()
	err := tilepush.LoadAllTileSetsFromDB(ctx)
	assert.NoError(t, err, "Failed to load tile sets before TestGetRandomTileSet")

	foundSets := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ts, err := tilepush.GetRandomTileSet()
		assert.NoError(t, err)
		assert.NotNil(t, ts)
		t.Logf("ts: %s", ts.Name)
		foundSets[ts.Name] = true
	}

	expectedCount := 22
	assert.Len(t, foundSets, expectedCount, "Expected to find all seeded tile sets")

	// 특정 콘셉트들이 포함되어 있는지 확인
	assert.True(t, foundSets["animals"])
	assert.True(t, foundSets["fruits"])
	assert.True(t, foundSets["concept_1"])
	assert.True(t, foundSets["concept_20"])
}
