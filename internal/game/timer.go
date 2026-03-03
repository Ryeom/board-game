package game

import (
	"sync"
	"time"
)

type TurnTimer struct {
	mu        sync.Mutex
	roomID    string
	duration  time.Duration
	timer     *time.Timer
	onExpire  func(roomID string)
	startedAt time.Time
	stopped   bool
}

func NewTurnTimer(roomID string, duration time.Duration, onExpire func(roomID string)) *TurnTimer {
	return &TurnTimer{
		roomID:   roomID,
		duration: duration,
		onExpire: onExpire,
		stopped:  true,
	}
}

func (t *TurnTimer) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopLocked()
	t.stopped = false
	t.startedAt = time.Now()
	t.timer = time.AfterFunc(t.duration, func() {
		t.mu.Lock()
		if t.stopped {
			t.mu.Unlock()
			return
		}
		t.stopped = true
		t.mu.Unlock()
		t.onExpire(t.roomID)
	})
}

func (t *TurnTimer) Reset() {
	t.Start()
}

func (t *TurnTimer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopLocked()
}

func (t *TurnTimer) stopLocked() {
	t.stopped = true
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
}

func (t *TurnTimer) Remaining() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped || t.timer == nil {
		return 0
	}
	elapsed := time.Since(t.startedAt)
	remaining := t.duration - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (t *TurnTimer) Duration() time.Duration {
	return t.duration
}
