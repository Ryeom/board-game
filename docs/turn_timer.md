# 턴 타이머 아키텍처

## 개요

게임모드별 고정 턴 제한 시간을 두고, 시간 초과 시 서버가 자동으로 랜덤 액션을 수행하는 시스템.

| 게임모드 | 턴 제한 시간 | 타임아웃 동작 |
|---------|------------|-------------|
| 하나비 | 60초 | 랜덤 카드 버리기 (힌트 만석 시 플레이) + end_turn |
| 타일푸시 | 30초 | 턴 스킵 (다음 플레이어로 넘김) |

## 컴포넌트 구조

```
┌─────────────────────────────────────────────────────────┐
│                     GameService                          │
│  - StartGame: 타이머 생성 + 시작                           │
│  - ProcessAction: 타이머 리셋                              │
│  - EndGame: 타이머 정지 (RemoveEngine 통해)                 │
│  - handleTimerExpired: 만료 콜백                           │
└──────────────┬───────────────────────────┬───────────────┘
               │                           │
    ┌──────────▼──────────┐    ┌───────────▼──────────┐
    │      Manager        │    │     TurnTimer         │
    │  engines: map[room] │    │  - time.AfterFunc     │
    │  timers:  map[room] │    │  - onExpire callback  │
    │  (sync.RWMutex)     │    │  - stopped flag       │
    └──────────┬──────────┘    │  (sync.Mutex)         │
               │               └──────────────────────┘
    ┌──────────▼──────────┐
    │    Engine (interface)│
    │  - GetTurnDuration() │
    │  - ExecuteForceAction│
    └─────────────────────┘
```

## 핵심 파일

| 파일 | 역할 |
|------|------|
| `internal/game/timer.go` | `TurnTimer` 구조체 — `time.AfterFunc` 기반 타이머 |
| `internal/game/engine_interface.go` | `Engine` 인터페이스에 `GetTurnDuration`, `ExecuteForceAction` 정의 |
| `internal/game/manager.go` | `timers` 맵으로 방별 타이머 관리, `RemoveEngine`에서 자동 정리 |
| `internal/game/hanabi/engine.go` | 하나비 `ExecuteForceAction` 구현 (discard/play + end_turn) |
| `internal/game/tilepush/engine.go` | 타일푸시 `ExecuteForceAction` 구현 (skip turn) |
| `internal/service/game_service.go` | 타이머 라이프사이클 관리 및 `handleTimerExpired` 콜백 |

## TurnTimer 설계

```go
type TurnTimer struct {
    mu        sync.Mutex
    roomID    string
    duration  time.Duration
    timer     *time.Timer       // time.AfterFunc 반환값
    onExpire  func(roomID string)
    startedAt time.Time
    stopped   bool              // Stop/Reset 후 콜백 실행 방지
}
```

### 동작 원리

- **Start/Reset**: 기존 타이머 정지 → `stopped = false` → `time.AfterFunc(duration, callback)` 호출.
- **Stop**: `stopped = true` → `timer.Stop()`.
- **콜백 실행**: `time.AfterFunc`의 고루틴에서 `stopped` 체크 후 `onExpire` 호출.

### `time.AfterFunc` 선택 이유

- 발동 시에만 고루틴 생성 (타이머 대기 중에는 고루틴 없음).
- `Reset`이 간단 — Stop 후 새로 생성.
- 콜백이 별도 고루틴에서 실행되므로 호출자 블로킹 없음.

## 동시성 및 데드락 방지

### 뮤텍스 계층

| 뮤텍스 | 위치 | 보호 대상 |
|--------|------|----------|
| `TurnTimer.mu` | `timer.go` | 타이머 상태 (stopped, timer, startedAt) |
| `Engine.mu` | 각 엔진 | 게임 상태 (CurrentState, PlayerHands 등) |
| `Manager.mu` | `manager.go` | engines/timers 맵 |

### 데드락 방지 규칙

1. **뮤텍스 중첩 금지**: `TurnTimer.mu`와 `Engine.mu`는 절대 동시에 잡지 않음.
2. **콜백 외부 실행**: 타이머 콜백(`onExpire`)은 `TurnTimer.mu`를 **해제한 후** 실행됨.
3. **원자적 ForceAction**: `ExecuteForceAction`은 `Engine.mu` 한 번만 잡고 auto-action + end_turn을 모두 처리.

### 콜백 실행 흐름

```
time.AfterFunc 고루틴:
  1. TurnTimer.mu.Lock()
  2. stopped 체크 → true이면 Unlock + return
  3. stopped = true
  4. TurnTimer.mu.Unlock()     ← 여기서 해제
  5. onExpire(roomID) 호출      ← 타이머 뮤텍스 없이 실행
     → GameService.handleTimerExpired()
       → engine.ExecuteForceAction()  ← Engine.mu 획득
```

## 경합 시나리오

### 시나리오 1: 플레이어 액션이 먼저

```
Player goroutine:              Timer goroutine:
  engine.HandleEvent()  ←  (대기)
  timer.Reset()
    → TurnTimer.mu.Lock()
    → stopped = true        ← 기존 타이머 무효화
    → timer.Stop()
    → 새 timer 생성
    → TurnTimer.mu.Unlock()
                               timer fires (이전 것)
                               TurnTimer.mu.Lock()
                               stopped == true → return  ← 실행 안됨
```

### 시나리오 2: 타이머가 먼저

```
Timer goroutine:               Player goroutine:
  onExpire() 호출
  engine.ExecuteForceAction()
    → Engine.mu.Lock()
    → 자동 액션 + end_turn
    → Engine.mu.Unlock()
                               engine.HandleEvent()
                                 → Engine.mu.Lock()
                                 → validateTurn()
                                 → "not your turn" 에러 ← 턴이 이미 넘어감
```

## 하나비 ExecuteForceAction 상세

```go
func (e *Engine) ExecuteForceAction() error {
    e.mu.Lock()
    defer e.mu.Unlock()

    // 1. 게임 종료/nil 체크
    // 2. 현재 플레이어의 핸드에서 랜덤 카드 인덱스 선택
    // 3. 힌트 토큰 상태에 따라 분기:
    //    - HintTokens >= 8 (만석) → handlePlayCard (페널티)
    //    - HintTokens < 8        → handleDiscardCard
    // 4. handleEndTurn() → 턴 넘김
    // 5. SetGameState → Redis 저장
    // 6. Broadcast("game.action.sync") → 클라이언트 동기화
}
```

**힌트 토큰 만석 시 play_card 폴백 이유**: 하나비 규칙상 힌트 토큰이 8개(최대)이면 버리기 불가. 유일한 대안은 카드 플레이. 이는 타임아웃에 대한 자연스러운 페널티가 됨 (잘못된 카드를 내면 미스 토큰 감소).

## 클라이언트 이벤트

| 이벤트 | 페이로드 | 시점 |
|--------|---------|------|
| `game.timer.started` | `{roomId, durationSecs}` | 게임 시작 시 |
| `game.timer.reset` | `{roomId, durationSecs}` | 매 턴 액션 후 |
| `game.timer.expired` | `{roomId}` | 타이머 발동 (auto-action 직전) |

모든 타이머 이벤트는 서버→클라이언트 단방향 (핸들러 등록 불필요).
