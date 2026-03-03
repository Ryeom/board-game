package game

import "time"

type Engine interface {
	StartGame()
	HandleEvent(event any) error
	EndGame()
	IsGameOver() bool
	GetTurnDuration() time.Duration // 게임모드별 턴 제한 시간 (0이면 타이머 비활성)
	ExecuteForceAction() error      // 타임아웃 시 자동 액션 (원자적 실행)
}

type State interface {
}
