package hanabi

import (
	"github.com/Ryeom/hanabi/util"
)

type CardSet []Card

type Card struct {
	Color      string
	Number     int
	KnowColor  bool
	KnowNumber bool
}

func createNewCardSet() CardSet {
	colors := []string{"green", "white", "yellow", "red", "blue"}
	numbers := map[int]int{1: 3, 2: 2, 3: 2, 4: 2, 5: 1}
	temp := CardSet{}
	for _, color := range colors {
		for cardNumber, cnt := range numbers {
			for i := 0; i < cnt; i++ {
				temp = append(temp, Card{Color: color, Number: cardNumber})
			}
		}
	}
	// 중복없는 난수 생성
	var t []int
	length := len(temp)
	index := 0
	for {
		n := util.RandomNumber(1, length+1)
		// 포함되지않으면 옮기고 0으로 변경
		if !util.IntContains(t, n) {
			t = append(t, n)
			index++
		}
		if index == length {
			break
		}
	}
	// 난수에 맞춰 카드배열
	set := make(CardSet, len(temp)) // 원본과 동일한 크기의 슬라이스 생성
	//copy(set, temp) // 복사안함
	for i, n := range t {
		set[n-1] = temp[i]
	}
	return set
}
