package game

type Engine interface {
	StartGame()
	HandleEvent(event any) error
}
