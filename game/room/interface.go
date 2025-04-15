package room

type GameEngine interface {
	StartGame() // <- 수정됨: *Room 파라미터 제거
	HandleEvent(event any) error
}
