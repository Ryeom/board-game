package game

type GameEngine interface {
	StartGame()
	HandleEvent(event any) error
}
