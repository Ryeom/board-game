package hanabi

import "fmt"

//var ActionExecution chan *Action

type Action struct {
	Target   string `json:"target"`
	Name     string `json:"name"`
	Response chan string
}

func init() {

}

func Initialize() {
	initializeRooms()
	//go InitializeInstructor()
}

// TODO : 게임 실행시 실행되는 것으로 변경

func InitializeInstructor() {
	ActionExecution := make(chan *Action)
	for {
		select {
		case act := <-ActionExecution:
			fmt.Println("Instructor command : ", *act)
			resp := execute(act)
			act.Response <- resp
		}
	}
}

func execute(act *Action) string {
	switch act.Target {
	case "attender":
		return attenderAction(act)
	case "game":
		return gameAction(act)
	case "room":
		return roomAction(act)

	}
	return "error"
}
