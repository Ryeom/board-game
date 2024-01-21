package hanabi

import (
	"fmt"
	"testing"
)

func TestCreateCard(t *testing.T) {
	c := createNewCardSet()
	fmt.Println(c)
	fmt.Println(len(c))
}