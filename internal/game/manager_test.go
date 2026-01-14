package game_test

import (
	"testing"

	"github.com/Ryeom/board-game/internal/game"
	"github.com/stretchr/testify/assert"
)

type MockEngine struct{}

func (m *MockEngine) StartGame()                  {}
func (m *MockEngine) HandleEvent(event any) error { return nil }
func (m *MockEngine) EndGame()                    {}
func (m *MockEngine) IsGameOver() bool            { return false }

func TestManager_AddAndGetEngine(t *testing.T) {
	manager := game.NewManager()
	engine := &MockEngine{}
	roomID := "room-1"


	_, ok := manager.GetEngine(roomID)
	assert.False(t, ok)

	// 엔진 추가
	manager.AddEngine(roomID, engine)

	// 엔진 조회
	retrieved, ok := manager.GetEngine(roomID)
	assert.True(t, ok)
	assert.Equal(t, engine, retrieved)
}

func TestManager_RemoveEngine(t *testing.T) {
	manager := game.NewManager()
	engine := &MockEngine{}
	roomID := "room-1"

	manager.AddEngine(roomID, engine)
	manager.RemoveEngine(roomID)

	_, ok := manager.GetEngine(roomID)
	assert.False(t, ok)
}

func TestManager_Concurrency(t *testing.T) {
	manager := game.NewManager()
	engine := &MockEngine{}
	roomID := "room-concurrent"

	go func() {
		for i := 0; i < 1000; i++ {
			manager.AddEngine(roomID, engine)
		}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			manager.GetEngine(roomID)
		}
	}()
}
