package http

import (
	"github.com/Ryeom/board-game/internal/auth"
	resp "github.com/Ryeom/board-game/internal/response"
	"github.com/Ryeom/board-game/server/ws"
	"github.com/labstack/echo/v4"
	"net/http"
)

func InitializeRouter(e *echo.Echo) {
	bg := e.Group("/board-game")
	{
		bg.GET("/healthCheck", healthCheck)
		bg.GET("/ws", ws.Websocket)

		authGroup := bg.Group("/auth")
		{
			authGroup.POST("/signup", SignUp)
			authGroup.POST("/login", Login)
		}

		apiGroup := bg.Group("/api")
		apiGroup.Use(auth.JWTMiddleware)
		{
			apiGroup.POST("/auth/logout", Logout)

			apiGroup.GET("/user/profile", GetUserProfile)
			apiGroup.PATCH("/user/profile", UpdateUserProfile)
			apiGroup.POST("/user/change-password", ChangePassword)

		}
	}

}

// @Summary Health Check
// @Description 서버 상태 확인
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} session.HttpResult
// @Router /board-game/healthCheck [get]
func healthCheck(c echo.Context) error {
	result := resp.HttpResult{
		Code:    "SUCCESS_HEALTH_CHECK",
		Message: "OK",
	}
	var param struct {
		Data interface{} `json:"data"`
	}
	if bindErr := c.Bind(&param); bindErr != nil {

	}
	o := map[string]interface{}{
		//"header": string(h),
		"header": c.Request().Header,
		//"body":   b,
		"body": param,
	}

	//r, err := json.Marshal(o)
	//if err != nil {
	//	fmt.Println(err)
	//}
	result.Data = o

	return c.JSON(http.StatusOK, result)
}
