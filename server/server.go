package server

import (
	"context"
	_ "github.com/Ryeom/board-game/docs" // swagger docs import
	"github.com/Ryeom/board-game/infra/db"
	"github.com/Ryeom/board-game/infra/mongo"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/auth"
	"github.com/Ryeom/board-game/internal/util"
	l "github.com/Ryeom/board-game/log"
	appHttp "github.com/Ryeom/board-game/server/http"
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

	e.HTTPErrorHandler = httpErrorHandler(e)

	e.GET("/swagger/*", echoSwagger.WrapHandler)
	redisutil.Initialize()
	db.Initialize()
	appHttp.InitializeRouter(e)
	auth.Initialize()
	mongo.Initialize()

	ctx := context.Background()
	ws.GlobalBroadcaster = ws.NewRedisBroadcaster(ctx)

}

func httpErrorHandler(e *echo.Echo) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {

	}
}
