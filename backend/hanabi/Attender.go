package hanabi

import (
	"github.com/gorilla/websocket"
)

type Attender struct {
	Id       string
	Ip       string
	Ready    bool
	HoldCard []Card
	Ws       *websocket.Conn
}

var Attenders map[string]*Attender

func attenderAction(act *Action) string {
	switch act.Name {
	case "ready-user":

	case "create":
		//return createAttender("10.70.222.112")
	}
	return "error"
}

// Id 입력 시 참여자 생성~

func CreateAttender(ip, id string, ws *websocket.Conn) string {
	a := getAttender(ip, id)
	if a == nil {
		a = &Attender{Ip: ip, Id: id, Ws: ws}
		Attenders[ip] = a
	}

	return "success"
}

func getAttender(ip, id string) *Attender {
	if Attenders[id] != nil {
		return Attenders[id]
	}
	return nil
}
