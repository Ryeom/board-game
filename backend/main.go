package main

import (
	"fmt"
	"github.com/Ryeom/hanabi/hanabi"
	l "github.com/Ryeom/hanabi/log"
	"github.com/Ryeom/hanabi/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"runtime"
)

func init() {

}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("start hanabi game")
	l.InitializeApplicationLog()

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(l.CreateCustomLogConfig()))

	hanabi.Initialize()
	server.Initialize(e)

	e.Logger.Fatal(e.Start(":8080"))
}
