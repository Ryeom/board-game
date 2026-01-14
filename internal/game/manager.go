package game

import "sync"

type Manager struct {
	mu      sync.RWMutex
	engines map[string]Engine
}

func NewManager() *Manager {
	return &Manager{
		engines: make(map[string]Engine),
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
}
