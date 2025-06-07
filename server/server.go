package server

import (
	"context"
	_ "github.com/Ryeom/board-game/docs" // swagger docs import
	"github.com/Ryeom/board-game/infra/db"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/util"
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/server/http"
	"github.com/Ryeom/board-game/server/ws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func Initialize(e *echo.Echo) {

	e.Validator = util.NewValidator()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(l.CreateCustomLogConfig()))

	e.GET("/swagger/*", echoSwagger.WrapHandler)
	redisutil.Initialize()
	db.Initialize()
	http.InitializeRouter(e)

	ctx := context.Background()
	ws.GlobalBroadcaster = ws.NewRedisBroadcaster(ctx)

}
