package game_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ryeom/board-game/internal/game"
	"github.com/stretchr/testify/assert"
)

func TestTurnTimer_StartWithNilCallback_DoesNotPanic(t *testing.T) {
	timer := game.NewTurnTimer("room-1", 5*time.Millisecond, nil)

	assert.NotPanics(t, func() {
		timer.Start()
		time.Sleep(10 * time.Millisecond)
	})
}

func TestTurnTimer_Stop_PreventsExpirationCallback(t *testing.T) {
	var called atomic.Int32
	timer := game.NewTurnTimer("room-1", 20*time.Millisecond, func(roomID string) {
		called.Add(1)
	})

	timer.Start()
	timer.Stop()
	time.Sleep(30 * time.Millisecond)

	assert.Equal(t, int32(0), called.Load())
}
