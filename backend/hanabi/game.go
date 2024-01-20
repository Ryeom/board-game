package hanabi

import "time"

type Game struct {
	Round int
	Log   []Progress
}

type Progress struct {
	RoomId     string
	GameId     string
	AttenderId string
	Round      int
	ActionCode string
	Message    string
	Timestamp  time.Time
}
