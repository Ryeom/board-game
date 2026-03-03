package game

import "sync"

type Manager struct {
	mu      sync.RWMutex
	engines map[string]Engine
	timers  map[string]*TurnTimer
}

func NewManager() *Manager {
	return &Manager{
		engines: make(map[string]Engine),
		timers:  make(map[string]*TurnTimer),
	}
}

func (m *Manager) AddEngine(roomID string, engine Engine) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.engines[roomID] = engine
}

func (m *Manager) GetEngine(roomID string) (Engine, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	engine, ok := m.engines[roomID]
	return engine, ok
}

func (m *Manager) RemoveEngine(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.engines, roomID)
	if timer, ok := m.timers[roomID]; ok {
		timer.Stop()
		delete(m.timers, roomID)
	}
}

func (m *Manager) SetTimer(roomID string, timer *TurnTimer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timers[roomID] = timer
}

func (m *Manager) GetTimer(roomID string) (*TurnTimer, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	timer, ok := m.timers[roomID]
	return timer, ok
}
