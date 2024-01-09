package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func init() {

}

func main() {
	fmt.Println("start hanabi game")
	router := gin.Default()

	router.Run(":8080")
}
