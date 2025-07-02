package ws

import (
	"context"
	"github.com/Ryeom/board-game/internal/user"
)

func HandleSystemPing(ctx context.Context, user *user.Session, event SocketEvent) {}

func HandleSystemError(ctx context.Context, user *user.Session, event SocketEvent) {}

func HandleSystemNotice(ctx context.Context, user *user.Session, event SocketEvent) {}

func HandleSystemSync(ctx context.Context, user *user.Session, event SocketEvent) {}
