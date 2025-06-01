package server

import (
	"context"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/server/http"
	"github.com/Ryeom/board-game/server/ws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func Initialize(e *echo.Echo) {
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(l.CreateCustomLogConfig()))
	redisutil.Initialize()
	http.InitializeRouter(e)
	ctx := context.Background()
	ws.GlobalBroadcaster = ws.NewRedisBroadcaster(ctx)

}
