package server

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func Initialize(e *echo.Echo) {
	apis := e.Group("/hanabi")
	{
		route(apis)
	}

}

func route(g *echo.Group) {
	g.GET("/healthCheck", healthCheck)

}

func healthCheck(c echo.Context) error {
	result := HttpResult{
		Code: 200,
		Msg:  "OK",
	}
	//b, _ := c.Request().GetBody()

	//h, err := json.Marshal(c.Request().Header)
	//if err != nil {
	//	fmt.Println(err)
	//}

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
