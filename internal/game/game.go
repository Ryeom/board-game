package game

type Mode string

const (
	ModeHanabi   Mode = "hanabi"
	Mode6Nimmt   Mode = "6nimmt"
	ModeTilePush Mode = "tile_push"
)

type Status string

const (
	StatusDefault Status = "default"
	StatusPlaying Status = "playing"
	StatusPaused  Status = "paused"
)
