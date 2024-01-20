package hanabi

import "github.com/Ryeom/hanabi/util"

type Attender struct {
	Id string
	Ip string
}

var Attenders map[string]*Attender

func CreateAttender() {
	a := Attender{}
	uuid := util.GetUUID()
	Attenders[uuid] = &a
}