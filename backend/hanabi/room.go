package hanabi

import (
	"github.com/Ryeom/hanabi/log"
	"github.com/Ryeom/hanabi/util"
)

const (
	WAITING = iota
	RUNNING
)
const (
	MaxAttender = 8
)

type Room struct {
	MaxAttender int
	Attender    []Attender
	Status      int
	Name        string
	Leader      string
}

var Rooms map[string]*Room

func initializeRooms() {
	Rooms = map[string]*Room{}
	log.Logger.Info("Finished create Room List")
}

func CreateRoom(constructor, roomName string) {
	r := Room{
		MaxAttender: MaxAttender,
		Status:      WAITING,
		Name:        roomName,
		//Attender:    []Attender,
		Leader: constructor,
	}
	uuid := util.GetUUID()
	Rooms[uuid] = &r
}

func exitRoom(targetId string) {

}
