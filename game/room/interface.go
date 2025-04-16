package room

type GameEngine interface {
	StartGame()
	HandleEvent(event any) error
}
