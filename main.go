package main

import (
	"fmt"
	_ "github.com/Ryeom/board-game/docs" // swagger docs import
	l "github.com/Ryeom/board-game/log"
	redis "github.com/Ryeom/board-game/redis"
	"github.com/Ryeom/board-game/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func init() {

}

func main() {
	fmt.Println("start board game")
	l.InitializeApplicationLog()

	redis.Initialize()
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(l.CreateCustomLogConfig()))
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	session.InitializeRouter(e)

	e.Logger.Fatal(e.Start(":8080"))
}
