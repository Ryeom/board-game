// room/attender.go
package room

type Attender struct {
	ID     string
	Name   string
	IsHost bool
}

func NewAttender(id, name string, isHost bool) *Attender {
	return &Attender{
		ID:     id,
		Name:   name,
		IsHost: isHost,
	}
}
