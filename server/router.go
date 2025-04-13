package server

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func Initialize(e *echo.Echo) {
	e.Static("/", "../public")
	apis := e.Group("/hanabi")
	{
		route(apis)
	}

	//http.Handle("/", http.FileServer(http.Dir("static")))
	//http.HandleFunc("/ws", socketHandler)
	//port := "8080"
	//if err := http.ListenAndServe(":"+port, nil); err != nil {
	//	log.Fatal(err)
	//}
}

func route(g *echo.Group) {
	g.GET("/healthCheck", healthCheck)
	g.GET("/ws", SocketHandler)
	//
	//g.GET()
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
