package hanabi

import (
	"fmt"
	"github.com/Ryeom/hanabi/util"
	"time"
)

type Game struct {
	Round      int
	Log        []Progress
	Attenders  []*Attender
	PublicCard CardSet
}

type Progress struct {
	RoomId     string
	GameId     string
	AttenderId string
	Round      int
	ActionCode string
	Message    string
	Timestamp  time.Time
}

func gameAction(act *Action) string {
	switch act.Target {

	case "start":
	case "end":
	case "next-turn":
	case "access":

	}
	return "erorr"
}
func start() {
	g := Game{
		PublicCard: createNewCardSet(),
		Round:      1,
		Attenders:  []*Attender{},
		Log:        []Progress{},
	}
	g.Log = append(g.Log, Progress{
		RoomId:     "",
		GameId:     util.GetUUID(),
		AttenderId: "",
		Round:      0,
		ActionCode: "",
		Message:    "게임이 시작되었습니다.",
		Timestamp:  time.Now(),
	})

	// 게임 진행 하는 코드 실행
	go runningGame(&g)
}

func runningGame(g *Game) { // 여기서 라운드 진행l
	ActionExecution := make(chan *Action)
	for {
		for {
			select {
			case act := <-ActionExecution:
				fmt.Println("Instructor command : ", *act)
				resp := execute(act)
				act.Response <- resp

				//case : // 카드고르는 시간 세어주어야함

			}
		}
	}
}
