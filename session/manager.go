// session/manager.go
package session

import (
	"sync"
)

var (
	sessions = make(map[string]*UserSession)
	mu       sync.RWMutex
)

func Register(user *UserSession) {
	mu.Lock()
	defer mu.Unlock()
	sessions[user.ID] = user
}

func Unregister(id string) {
	mu.Lock()
	defer mu.Unlock()
	delete(sessions, id)
}

func Get(id string) (*UserSession, bool) {
	mu.RLock()
	defer mu.RUnlock()
	user, ok := sessions[id]
	return user, ok
}

func GetAll() []*UserSession {
	mu.RLock()
	defer mu.RUnlock()
	list := make([]*UserSession, 0, len(sessions))
	for _, user := range sessions {
		list = append(list, user)
	}
	return list
}
