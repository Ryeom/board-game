package main

import (
	"fmt"
	"github.com/Ryeom/board-game/game"
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func init() {

}

func main() {
	fmt.Println("start board game")
	l.InitializeApplicationLog()

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(l.CreateCustomLogConfig()))

	game.Initialize()
	server.Initialize(e)

	e.Logger.Fatal(e.Start(":8080"))
}
