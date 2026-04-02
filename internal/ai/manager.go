package ai

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/Ryeom/board-game/log"
)

const (
	MinDelay = 2 * time.Second
	MaxDelay = 3 * time.Second
)

// ActionExecutor AI가 결정한 액션을 실행하는 함수 타입
type ActionExecutor func(ctx context.Context, roomID, aiPlayerID string, actionData map[string]any) error

// ActionDecider 현재 게임 상태를 보고 AI의 행동을 결정하는 함수 타입
type ActionDecider func(roomID, aiPlayerID string) (map[string]any, error)

// AIPlayerManager AI 플레이어의 턴 스케줄링을 담당한다.
type AIPlayerManager struct {
	mu            sync.Mutex
	pendingTimers map[string]*time.Timer // roomID → 대기 중인 타이머
	executor      ActionExecutor
}

func NewAIPlayerManager(executor ActionExecutor) *AIPlayerManager {
	return &AIPlayerManager{
		pendingTimers: make(map[string]*time.Timer),
		executor:      executor,
	}
}

// ScheduleAITurn AI 턴을 2~3초 딜레이 후 실행하도록 스케줄한다.
func (m *AIPlayerManager) ScheduleAITurn(roomID, aiPlayerID string, decider ActionDecider) {
	m.mu.Lock()
	// 기존 타이머가 있으면 취소
	if timer, ok := m.pendingTimers[roomID]; ok {
		timer.Stop()
	}

	delay := MinDelay + time.Duration(rand.Int63n(int64(MaxDelay-MinDelay)))
	log.Logger.Debugf("[AI] ScheduleAITurn room=%s player=%s delay=%v", roomID, aiPlayerID, delay)

	m.pendingTimers[roomID] = time.AfterFunc(delay, func() {
		m.mu.Lock()
		delete(m.pendingTimers, roomID)
		m.mu.Unlock()

		actionData, err := decider(roomID, aiPlayerID)
		if err != nil {
			log.Logger.Errorf("[AI] decider error room=%s player=%s: %v", roomID, aiPlayerID, err)
			return
		}

		if err := m.executor(context.Background(), roomID, aiPlayerID, actionData); err != nil {
			log.Logger.Errorf("[AI] executor error room=%s player=%s: %v", roomID, aiPlayerID, err)
		}
	})
	m.mu.Unlock()
}

// CancelRoom 방의 대기 중인 AI 타이머를 취소한다.
func (m *AIPlayerManager) CancelRoom(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if timer, ok := m.pendingTimers[roomID]; ok {
		timer.Stop()
		delete(m.pendingTimers, roomID)
		log.Logger.Debugf("[AI] CancelRoom room=%s", roomID)
	}
}

// Shutdown 모든 대기 타이머를 취소한다.
func (m *AIPlayerManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for roomID, timer := range m.pendingTimers {
		timer.Stop()
		delete(m.pendingTimers, roomID)
	}
	log.Logger.Debugf("[AI] Shutdown: all timers cancelled")
}
