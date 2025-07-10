package ws

import (
	"context"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
)

// HandleSystemPing 핑 체크에 대한 응답
func HandleSystemPing(ctx context.Context, user *user.Session, event SocketEvent) {
	sendResult(user, "pong", map[string]string{"message": "pong"}, resp.SuccessCodeSystemOK) // 변경: "system.ping" -> "pong"
}

func HandleSystemError(ctx context.Context, user *user.Session, event SocketEvent) {}

func HandleSystemNotice(ctx context.Context, user *user.Session, event SocketEvent) {}

func HandleSystemSync(ctx context.Context, user *user.Session, event SocketEvent) {}
