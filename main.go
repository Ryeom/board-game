package main

import (
	"fmt"
	_ "github.com/Ryeom/board-game/docs" // swagger docs import
	l "github.com/Ryeom/board-game/log"
	redis "github.com/Ryeom/board-game/redis"
	"github.com/Ryeom/board-game/session"
	"github.com/Ryeom/board-game/util"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func init() {
	/* 1. setting environment */
	err := util.SetEnv()
	if err != nil {
		panic(err)
	}
	/* 2. setting log  */
	err = l.InitializeApplicationLog()
	if err != nil {
		panic(err)
	}
	/* 3. setting config */
	err = util.SetConfig()
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Println("start board game")

	redis.Initialize()
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(l.CreateCustomLogConfig()))
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	session.InitializeRouter(e)

	e.Logger.Fatal(e.Start(":8080"))
}
