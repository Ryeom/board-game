package server

import (
	"context"
	_ "github.com/Ryeom/board-game/docs" // swagger docs import
	"github.com/Ryeom/board-game/infra/db"
	"github.com/Ryeom/board-game/infra/mongo"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/auth"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/internal/util"
	l "github.com/Ryeom/board-game/log"
	appHttp "github.com/Ryeom/board-game/server/http"
	"github.com/Ryeom/board-game/server/ws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"net/http"
	"time"
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

	// Initialize broadcaster
	broadcaster := ws.NewRedisBroadcaster(ctx)
	ws.GlobalBroadcaster = broadcaster

	broadcaster.SetSessionGetter(func(socketID string) (*user.Session, bool) {
		val, ok := ws.ActiveSessions().Load(socketID)
		if !ok {
			return nil, false
		}
		session, typeOk := val.(*user.Session)
		return session, typeOk
	})

}
func httpErrorHandler(e *echo.Echo) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		var code string
		var statusCode int = http.StatusInternalServerError
		var originalErr error = err

		httpErr, ok := err.(*echo.HTTPError)
		if ok {
			statusCode = httpErr.Code

			switch statusCode {
			case http.StatusBadRequest:
				code = resp.ErrorCodeDefaultBadRequest
			case http.StatusUnauthorized:
				code = resp.ErrorCodeDefaultUnauthorized
			case http.StatusForbidden:
				code = resp.ErrorCodeDefaultForbidden
			case http.StatusNotFound:
				code = resp.ErrorCodeDefaultNotFound
			default:
				code = resp.ErrorCodeDefaultInternalServerError
			}
			originalErr = httpErr.Internal
		} else {
			// 2. 그 외 일반적인 Go error 또는 panic으로부터 복구된 에러
			code = resp.ErrorCodeDefaultInternalServerError
			statusCode = http.StatusInternalServerError
		}

		response := resp.Fail(code, "ko")
		response.Timestamp = time.Now()

		if response.Error != nil {
			response.Error.HttpStatusCode = statusCode
		} else {
			response.Status = "error"
			response.Message = "알 수 없는 오류가 발생했습니다."
			response.Code = code
		}

		if originalErr != nil {
			l.Logger.Errorf("HTTP Error: Path=%s, Status=%d, Code=%s, Message=%s, OriginalError=%v",
				c.Path(), statusCode, code, response.Message, originalErr)
		} else {
			l.Logger.Errorf("HTTP Error: Path=%s, Status=%d, Code=%s, Message=%s",
				c.Path(), statusCode, code, response.Message)
		}

		if !c.Response().Committed {
			if err := c.JSON(statusCode, response); err != nil {
				l.Logger.Error(err)
			}
		}
	}
}
