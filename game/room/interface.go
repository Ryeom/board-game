package room

import "github.com/Ryeom/board-game/game/room/types"

type GameEngine interface {
	StartGame(r *Room)
	HandleEvent(r *Room, event types.Event) error
	// 필요 시 추후 ValidateMove, OnTurnEnd 등 확장 가능
}
