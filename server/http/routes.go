package http

import (
	"github.com/Ryeom/board-game/server/ws"
	"github.com/labstack/echo/v4"
	"net/http"
)

func InitializeRouter(e *echo.Echo) {
	bg := e.Group("/board-game")
	{
		bg.GET("/healthCheck", healthCheck)
		bg.GET("/ws", ws.Websocket)

		bg.GET("/api/rooms", GetRoomList)
		bg.POST("/api/rooms", CreateRoom)
		bg.PATCH("/api/rooms/:roomId", UpdateRoom)
		bg.DELETE("/api/rooms/:roomId", DeleteRoom)
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
	result := HttpResult{
		Code: 200,
		Msg:  "OK",
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

type HttpResult struct {
	Code interface{} `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}
