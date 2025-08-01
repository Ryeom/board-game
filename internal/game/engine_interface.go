package game

type Engine interface {
	StartGame()
	HandleEvent(event any) error
	EndGame()
	IsGameOver() bool
}

type State interface {
}
