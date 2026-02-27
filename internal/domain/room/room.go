package room

import (
	"context"
	"errors"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/game"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/util"
	"time"
)

type Room struct {
	ID            string          `json:"id"`
	RoomName      string          `json:"roomName"`
	Host          string          `json:"host"` // 방장
	Players       []string        `json:"players"`
	ReadyPlayers  map[string]bool `json:"readyPlayers"`
	Password      string          `json:"-"`
	MaxPlayers    int             `json:"maxPlayers"`
	GameMode      game.Mode       `json:"gameMode"`
	IsGameStarted bool            `json:"isGameStarted"`
	CreatedAt     time.Time       `json:"createdAt"`
}

func CreateRoom(ctx context.Context, roomID string, hostID string, roomName string, password string, maxPlayers int) (*Room, error) { // 인자 추가
	hashedPassword := ""
	if password != "" {
		var err error
		hashedPassword, err = util.HashPassword(password) //
		if err != nil {
			return nil, err
		}
	}

	if maxPlayers < 2 { // TODO 게임 모드별 최대 인원 제한
		maxPlayers = 2
	}
	r := &Room{
		ID:            roomID,
		RoomName:      roomName,
		Host:          hostID,
		Players:       []string{hostID},
		ReadyPlayers:  make(map[string]bool),
		Password:      hashedPassword,
		MaxPlayers:    maxPlayers,
		GameMode:      game.ModeHanabi,
		IsGameStarted: false,
		CreatedAt:     time.Now(),
	}
	if err := r.Save(); err != nil {
		return nil, err
	}
	return r, nil
}

func GetRoom(ctx context.Context, roomID string) (*Room, bool) {
	var r Room
	ok := redisutil.GetJSON("room", "room:"+roomID, &r)
	return &r, ok
}

func DeleteRoom(ctx context.Context, roomID string) error {
	rdb := redisutil.Client["room"]
	if rdb == nil {
		return errors.New(resp.ErrorCodeRoomDeleteFailed)
	}
	return redisutil.Delete("room", "room:"+roomID)
}

func ListRooms(ctx context.Context) []*Room {
	rdb := redisutil.Client["room"]
	if rdb == nil {
		return nil
	}
	keys, err := rdb.Keys(ctx, "room:*").Result()
	if err != nil {
		return nil
	}

	var rooms []*Room
	for _, key := range keys {
		var r Room
		if ok := redisutil.GetJSON("room", key, &r); ok {
			rooms = append(rooms, &r)
		}
	}
	return rooms
}

func (r *Room) Save() error {
	redisutil.SaveJSON("room", "room:"+r.ID, r, 0)
	return nil
}
func (r *Room) Join(ctx context.Context, userID string, password string) (bool, error) {
	// 1. 방 참여 인원 제한 확인
	if len(r.Players) >= r.MaxPlayers {
		return false, errors.New(resp.ErrorCodeRoomFull)
	}

	if r.IsGameStarted {
		return false, errors.New(resp.ErrorCodeGameAlreadyStarted)
	}

	// 2. 비밀번호가 설정된 방인 경우, 비밀번호 검증
	if r.Password != "" {
		if !util.CheckPasswordHash(password, r.Password) {
			return false, errors.New(resp.ErrorCodeRoomWrongPassword)
		}
	}

	// 3. 이미 참여 중인지 확인
	for _, p := range r.Players {
		if p == userID {
			return true, nil // 이미 참여 중
		}
	}

	// 4. 플레이어 추가 및 레디 상태 초기화
	r.Players = append(r.Players, userID)
	r.ResetReady()
	if err := r.Save(); err != nil {
		return false, errors.New(resp.ErrorCodeRoomJoinFailed)
	}
	return true, nil
}

// ToggleReady 플레이어의 레디 상태를 토글
func (r *Room) ToggleReady(userID string) bool {
	if r.ReadyPlayers == nil {
		r.ReadyPlayers = make(map[string]bool)
	}
	r.ReadyPlayers[userID] = !r.ReadyPlayers[userID]
	return r.ReadyPlayers[userID]
}

// AllPlayersReady 모든 플레이어가 레디 상태인지 확인
func (r *Room) AllPlayersReady() bool {
	if len(r.Players) < 2 {
		return false
	}
	for _, pid := range r.Players {
		if !r.ReadyPlayers[pid] {
			return false
		}
	}
	return true
}

// ResetReady 모든 플레이어의 레디 상태 초기화
func (r *Room) ResetReady() {
	r.ReadyPlayers = make(map[string]bool)
}
