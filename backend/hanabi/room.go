package hanabi

import "time"

type Room struct {
	ID        string
	Players   []*Attender
	Game      *Game
	State     *State
	CreatedAt time.Time
}
