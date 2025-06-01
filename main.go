package main

import (
	_ "github.com/Ryeom/board-game/docs" // swagger docs import
	"github.com/Ryeom/board-game/internal/util"
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/room"
	"github.com/Ryeom/board-game/server"
	"github.com/labstack/echo/v4"
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
	room.Initialize()
	e := echo.New()
	e.GET("/swagger/*", echoSwagger.WrapHandler)
	server.Initialize(e)

	e.Logger.Fatal(e.Start(":8080"))
}
