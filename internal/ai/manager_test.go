package ai

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Ryeom/board-game/log"
	"github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	log.Logger = logging.MustGetLogger("test")
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	logging.SetBackend(backend)
	os.Exit(m.Run())
}

func TestScheduleAITurn_ExecutesAfterDelay(t *testing.T) {
	var mu sync.Mutex
	var executed bool
	var capturedRoom, capturedPlayer string
	var capturedAction map[string]any

	executor := func(_ context.Context, roomID, aiPlayerID string, actionData map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		executed = true
		capturedRoom = roomID
		capturedPlayer = aiPlayerID
		capturedAction = actionData
		return nil
	}

	mgr := NewAIPlayerManager(executor)

	decider := func(roomID, aiPlayerID string) (map[string]any, error) {
		return map[string]any{"actionType": "play_card", "cardIndex": float64(0)}, nil
	}

	mgr.ScheduleAITurn("room1", "ai_1", decider)

	// 아직 실행 안 됨
	time.Sleep(500 * time.Millisecond)
	mu.Lock()
	assert.False(t, executed)
	mu.Unlock()

	// 3.5초 후에는 실행되어야 함
	time.Sleep(3 * time.Second)
	mu.Lock()
	require.True(t, executed)
	assert.Equal(t, "room1", capturedRoom)
	assert.Equal(t, "ai_1", capturedPlayer)
	assert.Equal(t, "play_card", capturedAction["actionType"])
	mu.Unlock()
}

func TestCancelRoom_StopsTimer(t *testing.T) {
	var mu sync.Mutex
	var executed bool

	executor := func(_ context.Context, _, _ string, _ map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		executed = true
		return nil
	}

	mgr := NewAIPlayerManager(executor)

	decider := func(_, _ string) (map[string]any, error) {
		return map[string]any{"actionType": "discard"}, nil
	}

	mgr.ScheduleAITurn("room1", "ai_1", decider)
	mgr.CancelRoom("room1")

	time.Sleep(4 * time.Second)
	mu.Lock()
	assert.False(t, executed)
	mu.Unlock()
}

func TestShutdown_CancelsAllTimers(t *testing.T) {
	var mu sync.Mutex
	execCount := 0

	executor := func(_ context.Context, _, _ string, _ map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		execCount++
		return nil
	}

	mgr := NewAIPlayerManager(executor)

	decider := func(_, _ string) (map[string]any, error) {
		return map[string]any{}, nil
	}

	mgr.ScheduleAITurn("room1", "ai_1", decider)
	mgr.ScheduleAITurn("room2", "ai_2", decider)
	mgr.Shutdown()

	time.Sleep(4 * time.Second)
	mu.Lock()
	assert.Equal(t, 0, execCount)
	mu.Unlock()
}

func TestScheduleAITurn_ReplacesPrevious(t *testing.T) {
	var mu sync.Mutex
	var capturedPlayer string
	execCount := 0

	executor := func(_ context.Context, _, aiPlayerID string, _ map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		capturedPlayer = aiPlayerID
		execCount++
		return nil
	}

	mgr := NewAIPlayerManager(executor)

	decider := func(_, aiPlayerID string) (map[string]any, error) {
		return map[string]any{"player": aiPlayerID}, nil
	}

	// 같은 방에 두 번 스케줄 → 첫 번째는 취소되어야 함
	mgr.ScheduleAITurn("room1", "ai_1", decider)
	mgr.ScheduleAITurn("room1", "ai_2", decider)

	time.Sleep(4 * time.Second)
	mu.Lock()
	assert.Equal(t, 1, execCount)
	assert.Equal(t, "ai_2", capturedPlayer)
	mu.Unlock()
}
