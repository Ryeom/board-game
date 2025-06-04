package main

import (
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/room"
	"github.com/Ryeom/board-game/server"
	"github.com/labstack/echo/v4"
)

func init() {
	/* 1. setting environment */
	err := server.SetEnv()
	if err != nil {
		panic(err)
	}
	/* 2. setting log  */
	err = l.InitializeApplicationLog()
	if err != nil {
		panic(err)
	}
	/* 3. setting config */
	err = server.SetConfig()
	if err != nil {
		panic(err)
	}
}

func main() {
	room.Initialize()
	e := echo.New()
	server.Initialize(e)
	e.Logger.Fatal(e.Start(":8080"))
}
