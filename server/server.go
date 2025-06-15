package server

import (
	"context"
	"errors"
	"fmt"
	_ "github.com/Ryeom/board-game/docs" // swagger docs import
	"github.com/Ryeom/board-game/infra/db"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	appErrors "github.com/Ryeom/board-game/internal/errors"
	"github.com/Ryeom/board-game/internal/util"
	l "github.com/Ryeom/board-game/log"
	appHttp "github.com/Ryeom/board-game/server/http"
	"github.com/Ryeom/board-game/server/ws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"net/http"
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

	ctx := context.Background()
	ws.GlobalBroadcaster = ws.NewRedisBroadcaster(ctx)

}

func httpErrorHandler(e *echo.Echo) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		var appErr *appErrors.AppError
		if errors.As(err, &appErr) {
			response := appHttp.NewErrorResponse(appErr)
			if !c.Response().Committed {
				if err := c.JSON(appErr.Status, response); err != nil {
					e.Logger.Error(err)
				}
			}
			if appErr.Err != nil {
				e.Logger.Errorf("HTTP Error (AppError): Status=%d, Code=%s, Message=%s, OriginalError=%v, Path=%s",
					appErr.Status, appErr.Code, appErr.Message, appErr.Err, c.Path())
			} else {
				e.Logger.Errorf("HTTP Error (AppError): Status=%d, Code=%s, Message=%s, Path=%s",
					appErr.Status, appErr.Code, appErr.Message, c.Path())
			}
			return
		}

		code := http.StatusInternalServerError
		message := "알 수 없는 서버 오류가 발생했습니다."
		httpErr, ok := err.(*echo.HTTPError)
		if ok {
			code = httpErr.Code
			if httpErr.Message != nil {
				message = fmt.Sprint(httpErr.Message)
			}
		} else {
			e.Logger.Errorf("HTTP Error (Generic): %v, Path=%s", err, c.Path())
		}

		response := appHttp.NewErrorResponse(appErrors.InternalServerError(message, err))
		response.Code = code
		response.Message = message

		if !c.Response().Committed {
			if err := c.JSON(code, response); err != nil {
				e.Logger.Error(err)
			}
		}
	}
}
