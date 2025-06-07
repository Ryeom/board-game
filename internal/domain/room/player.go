package room

type Player interface {
	GetID() string
	GetName() string
	IsHostUser() bool
}
