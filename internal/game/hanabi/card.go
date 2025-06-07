package hanabi

type Color string

const (
	Red    Color = "red"
	Green  Color = "green"
	Blue   Color = "blue"
	Yellow Color = "yellow"
	White  Color = "white"
)

type Card struct {
	Color       Color `json:"color"`
	Number      int   `json:"number"`
	ColorKnown  bool  `json:"colorKnown"`
	NumberKnown bool  `json:"numberKnown"`
}
