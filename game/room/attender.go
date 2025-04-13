package room

import (
	"github.com/Ryeom/board-game/game/hanabi"
	"time"
)

/*	플레이어 정보, 손 패, 힌트 상태 등 */
type Attender struct {
	ID       string         `json:"id"`       // WebSocket ID 또는 고유 UUID
	Name     string         `json:"name"`     // 닉네임
	Hand     []*hanabi.Card `json:"hand"`     // 손패 (자신은 볼 수 없고, 다른 사람이 보는 정보용)
	IsHost   bool           `json:"isHost"`   // 방장 여부
	JoinedAt int64          `json:"joinedAt"` // 입장 시간 (선착순 정렬용)
}

func NewAttender(id string, name string, isHost bool) *Attender {
	return &Attender{
		ID:       id,
		Name:     name,
		IsHost:   isHost,
		Hand:     []*hanabi.Card{},
		JoinedAt: time.Now().Unix(),
	}
}
