// room/types.go
package room

import (
	"context"
	"encoding/json"
	"fmt"
	redisutil "github.com/Ryeom/board-game/redis"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"log"
	"sync"
	"time"
)

type Player interface {
	GetID() string
	GetName() string
	IsHostUser() bool
}

type Manager interface {
	CreateRoom(ctx context.Context, roomID string, host Player) *Room
	GetRoom(ctx context.Context, roomID string) (*Room, bool)
	DeleteRoom(ctx context.Context, roomID string)
	ListRooms(ctx context.Context) []*Room
	SaveRoom(ctx context.Context, room *Room) error
	JoinRoom(ctx context.Context, roomID string, userID string) (*Room, bool)
}

type InMemoryManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		rooms: make(map[string]*Room),
	}
}

func (m *InMemoryManager) CreateRoom(ctx context.Context, roomID string, host Player) *Room {
	r := &Room{
		ID:        roomID,
		Host:      host.GetID(),
		Players:   []string{host.GetID()},
		GameMode:  GameModeHanabi,
		CreatedAt: time.Now(),
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[roomID] = r
	return r
}

func (m *InMemoryManager) GetRoom(ctx context.Context, roomID string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	return r, ok
}

func (m *InMemoryManager) DeleteRoom(ctx context.Context, roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, roomID)
}

func (m *InMemoryManager) ListRooms(ctx context.Context) []*Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		list = append(list, r)
	}
	fmt.Println("<UNK>", list)
	return list
}

func (m *InMemoryManager) SaveRoom(ctx context.Context, room *Room) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[room.ID] = room
	return nil
}

func (m *InMemoryManager) JoinRoom(ctx context.Context, roomID string, userID string) (*Room, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, false
	}
	for _, p := range r.Players {
		if p == userID {
			return r, true
		}
	}
	r.Players = append(r.Players, userID)
	return r, true
}

type RedisManager struct {
	Client *redis.Client
}

func NewRedisManager() *RedisManager {
	addr := viper.GetString("redis.addr")
	c, err := redisutil.CreateClient(addr, "", viper.GetInt("redis.room-index"))
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			keys, err := c.Keys(ctx, "room:*").Result()
			if err != nil {
				log.Println(err)
			}
			for _, key := range keys {
				fmt.Println(key)
			}
			fmt.Println("-----------------------------")
			cancel()
			time.Sleep(3 * time.Second)
		}
	}()
	return &RedisManager{
		Client: c,
	}
}

func (r *RedisManager) CreateRoom(ctx context.Context, roomID string, host Player) *Room {
	room := &Room{
		ID:        roomID,
		Host:      host.GetID(),
		Players:   []string{host.GetID()},
		GameMode:  GameModeHanabi,
		CreatedAt: time.Now(),
	}
	if err := r.SaveRoom(ctx, room); err != nil {
		log.Printf("failed to save room: %v", err)
	}
	return room
}

func (r *RedisManager) GetRoom(ctx context.Context, roomID string) (*Room, bool) {
	val, err := r.Client.Get(ctx, "room:"+roomID).Result()
	if err != nil {
		return nil, false
	}
	var room Room
	if err := json.Unmarshal([]byte(val), &room); err != nil {
		return nil, false
	}
	return &room, true
}

func (r *RedisManager) DeleteRoom(ctx context.Context, roomID string) {
	r.Client.Del(ctx, "room:"+roomID)
}

func (r *RedisManager) ListRooms(ctx context.Context) []*Room {
	keys, err := r.Client.Keys(ctx, "room:*").Result()
	if err != nil {
		return nil
	}
	var rooms []*Room
	for _, key := range keys {
		val, err := r.Client.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var room Room
		if err := json.Unmarshal([]byte(val), &room); err == nil {
			rooms = append(rooms, &room)
		}
	}
	return rooms
}

func (r *RedisManager) SaveRoom(ctx context.Context, room *Room) error {
	b, err := json.Marshal(room)
	if err != nil {
		return err
	}
	err = r.Client.Set(ctx, room.ID, b, 0).Err()
	if err != nil {
		fmt.Println("<UNK>", err)
	}
	return nil
}

func (r *RedisManager) JoinRoom(ctx context.Context, roomID string, userID string) (*Room, bool) {
	rm, ok := r.GetRoom(ctx, roomID)
	if !ok {
		return nil, false
	}
	for _, p := range rm.Players {
		if p == userID {
			return rm, true
		}
	}
	rm.Players = append(rm.Players, userID)
	if err := r.SaveRoom(ctx, rm); err != nil {
		return nil, false
	}
	return rm, true
}
