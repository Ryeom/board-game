package hanabi

import "fmt"

var ActionExecution chan *Action

type Action struct {
	Action string `json:"action"`

	Response chan string
}

func init() {

}

func Initialize() {
	initializeRooms()
	go InitializeInstructor()
}

func InitializeInstructor() {
	ActionExecution = make(chan *Action)
	for {
		select {
		case act := <-ActionExecution:
			fmt.Println("Instructor command : ", *act)
			act.Response <- "bye bye"
		}
	}
}
