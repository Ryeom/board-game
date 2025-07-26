package hanabi

import "time"

func randInt(min, max int) int {
	return min + int(time.Now().UnixNano())%(max-min)
}
