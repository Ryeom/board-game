package ws

import (
	"context"
	"encoding/json"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
)

func HandleDefault(ctx context.Context, u *user.Session, event SocketEvent) {
	eventJSON, _ := json.Marshal(event)
	log.Logger.Warningf(
		"[UNKNOWN_EVENT] type=%s userID=%s userName=%s roomID=%s ip=%s ua=%s event=%s",
		event.Type,
		u.ID,
		u.Name,
		u.RoomID,
		u.IP,
		u.UserAgent,
		string(eventJSON),
	)
	sendError(u, resp.ErrorCodeWSUnknownEvent)
}
