package hanabi

/*	플레이어 정보, 손 패, 힌트 상태 등 */
type Attender struct {
	ID     string
	Name   string
	Hand   []*Card
	Hints  []Hint
	IsHost bool
}
