package ws

import (
	"context"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
)

func HandleSystemPing(ctx context.Context, u *user.Session, event SocketEvent) {
	_ = u.Conn.WriteJSON(map[string]string{
		"type":    "pong",
		"message": "pong",
	})
}

func HandleSystemError(ctx context.Context, u *user.Session, event SocketEvent) {
	log.Logger.Errorf("Client reported system error. UserID: %s, Data: %+v", u.ID, event.Data)
	sendResult(u, event.Type, nil, resp.SuccessCodeSystemErrorReceived) // 성공 메시지 코드
}

func HandleSystemNotice(ctx context.Context, u *user.Session, event SocketEvent) {
	sendError(u, resp.ErrorCodeSystemFeatureNotImplemented)
}

func HandleSystemSync(ctx context.Context, u *user.Session, event SocketEvent) {
	sendError(u, resp.ErrorCodeSystemFeatureNotImplemented)
}
