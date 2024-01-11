package hanabi

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
}

var Rooms []*Room

func InitializeRooms() {
	Rooms = []*Room{}
}

func CreateRoom() {

}

func exitRoom(targetId string) {

}
