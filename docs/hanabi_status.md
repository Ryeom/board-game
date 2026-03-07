# 하나비 구현 현황

## 구현된 기능

### 핵심 게임 로직 (`internal/game/hanabi`)
- **게임 상태 (`State` 구조체)**
    - `Fireworks`(색상별 진행도), `HintTokens`(8개), `MissTokens`(3개) 추적.
    - `Deck` 생성 및 셔플 관리.
    - `PlayerHands` 관리 (플레이어 수에 따라 4~5장 분배).
    - **시야 마스킹**: `GetPlayerView`는 힌트를 통해 명시적으로 "알려진" 경우가 아니면 플레이어 자신의 카드 정보(색상/숫자)를 올바르게 숨김.

- **게임 액션**
    - **`give_hint`**:
        - 색상 또는 숫자로 힌트 제공 및 마킹.
        - 힌트 토큰 1개 소모.
        - 대상 카드의 `ColorKnown` / `NumberKnown` 플래그 업데이트.
        - 자기 자신에게 힌트 불가.
        - 매칭 카드 0장인 경우 빈 힌트 방지 (에러 반환).
        - 힌트 토큰 0개일 때 힌트 불가.
    - **`play_card`**:
        - 카드 인덱스 유효성 검사.
        - 카드가 현재 불꽃놀이 순서에 맞는지 확인.
        - **성공**: 불꽃놀이 점수 업데이트. '5'를 냈을 경우 힌트 토큰 1개 회복 (보너스 룰).
        - **실패**: 미스 토큰 1개 차감. 미스 토큰이 0 이하가 되면 `GameOver = true` 설정.
        - **드로우**: 덱에서 자동으로 새 카드를 가져옴.
    - **`discard`**:
        - 선택한 카드를 `DiscardPile`에 버림.
        - 힌트 토큰 1개 회복.
        - 새 카드를 드로우.
        - 힌트 토큰이 8개(최대)일 때 버리기 불가.
    - **`end_turn`**:
        - `TurnIndex` 진행.
        - 게임 종료를 위한 덱 소진 조건 확인 (마지막 라운드 로직).

- **턴 타이머** (`internal/game/timer.go`)
    - 하나비 턴 제한 시간: 60초.
    - 타임아웃 시 자동 랜덤 액션 수행 (`ExecuteForceAction`).
    - 힌트 토큰이 만석이 아니면 랜덤 카드 버리기, 만석이면 랜덤 카드 플레이.
    - auto-action 후 자동 `end_turn` (원자적 실행).
    - 클라이언트에 `game.timer.started`, `game.timer.reset`, `game.timer.expired` 이벤트 전송.

- **게임 종료 (`EndGame`)**
    - 최종 점수 계산 (`FinalScore`).
    - 종료 사유 설정: `perfect` (25점), `miss_depleted` (미스 토큰 소진), `deck_exhausted` (덱 소진).
    - `game.end` 이벤트로 플레이어별 뷰 브로드캐스트.

### 서버 및 인프라
- **웹소켓 이벤트**: `game.start`, `game.action`, `game.end`, `game.sync`, `game.info`.
- **타이머 이벤트** (서버→클라이언트): `game.timer.started`, `game.timer.reset`, `game.timer.expired`.
- **구조**: `Engine` 인터페이스 → `Manager` → `GameService` 아키텍처.
- **동시성**: Engine에 `sync.Mutex`, WebSocket Write에 `WriteMutex`, Manager에 `sync.RWMutex`.
- **재접속**: 게임 중 연결 끊김 시 세션/방 상태 복원.
- **레디 체크**: 모든 플레이어가 준비 완료해야 게임 시작 가능.

## 게임 규칙 적용 상태 체크리스트

`docs/game_rule/hanabi.md` 기준 구현 여부 점검 결과.

### 구성물 및 준비
- [x] **카드 구성**: 색상별 1(3), 2(2), 3(2), 4(2), 5(1) (`GenerateDeck`)
- [x] **토큰 설정**: 힌트 8개, 오답 3개 (`NewState`)
- [x] **카드 분배**: 인원수에 따른 4/5장 분배 (`DealInitialCards`)
- [x] **정보 숨김**: 자신의 카드 내용 숨김 처리 (`GetPlayerView`)

### 행동 A. 정보(힌트) 주기
- [x] **기본 기능**: 색상 또는 숫자로 힌트 제공 및 마킹
- [x] **비용**: 힌트 토큰 1개 소모
- [x] **유효성 검사**: 빈 힌트 방지 (매칭 카드 0장이면 에러 반환)
- [x] **자기 자신 힌트 금지**
- [x] **토큰 부족 시 에러 반환**

### 행동 B. 카드 내려놓기 (등록)
- [x] **성공/실패 판정**: 올바른 순서면 성공, 아니면 실패
- [x] **보너스**: 숫자 '5' 완성 시 힌트 토큰 +1
- [x] **실패 페널티**: 미스 토큰 감소

### 행동 C. 카드 버리기
- [x] **기본 기능**: 카드 버리기 및 드로우
- [x] **조건 제약**: 힌트 토큰이 꽉 찼을(8개) 경우 버리기 행동 불가

### 게임 종료 및 승패
- [x] **승리 조건**: 25점 완성 시 즉시 종료
- [x] **패배 조건**: 미스 토큰 소진 시 즉시 패배
- [x] **덱 소진 종료**: 덱 소진 후 마지막 턴 진행 로직
- [x] **최종 점수/종료 사유**: `FinalScore`, `EndReason` 포함하여 브로드캐스트

### 턴 타이머
- [x] **턴 제한 시간**: 60초 (게임모드별 고정)
- [x] **타임아웃 자동 액션**: 랜덤 카드 버리기 (힌트 만석 시 플레이)
- [x] **원자적 실행**: auto-action + end_turn 한 번의 Lock으로 처리
- [x] **경합 방지**: 타이머와 플레이어 액션 간 안전한 경합 처리

## 테스트 현황

총 **17개** 테스트 (`internal/game/hanabi/engine_test.go`):

| 테스트명 | 검증 내용 |
|---------|----------|
| `TestHandlePlayCard_VictoryCondition` | 25점 달성 시 GameOver 설정 |
| `TestValidateTurn_WrongPlayer` | 다른 플레이어 턴에 행동 시 에러 |
| `TestValidateTurn_CorrectPlayer` | 본인 턴에 정상 행동 |
| `TestGiveHint_ToSelf` | 자기 자신에게 힌트 금지 |
| `TestGiveHint_EmptyHint` | 매칭 카드 0장인 힌트 방지 |
| `TestGiveHint_NoTokens` | 힌트 토큰 0개일 때 에러 |
| `TestGiveHint_WrongTurn` | 다른 플레이어 턴에 힌트 에러 |
| `TestDiscard_WhenHintTokensFull` | 힌트 토큰 만석 시 버리기 불가 |
| `TestEndGame_PerfectScore` | 만점 종료 사유 `perfect` |
| `TestEndGame_MissDepleted` | 미스 소진 종료 사유 `miss_depleted` |
| `TestEndGame_DeckExhausted` | 덱 소진 종료 사유 `deck_exhausted` |
| `TestGameFlow_FullCycle` | 시작→힌트→턴종료→버리기 통합 흐름 |
| `TestGameFlow_MissTokenDepletion` | 미스 토큰 소진으로 게임 종료 전체 흐름 |
| `TestExecuteForceAction_Discard` | 타임아웃 시 랜덤 카드 버리기 |
| `TestExecuteForceAction_PlayCard_WhenHintTokensFull` | 힌트 만석 시 play_card 폴백 |
| `TestExecuteForceAction_GameOver_NoOp` | 게임 종료 상태에서 no-op |
| `TestDiscard_WhenHintTokensNotFull` | 힌트 토큰 미만석 시 정상 버리기 |

## 로드맵

| ID | 작업 분류 | 상세 내용 | 상태 |
| :--- | :--- | :--- | :--- |
| **TASK-001** | 서버 와이어링 | `events_game.go`에서 `hanabi.NewEngine` 초기화 연결 | [x] |
| **TASK-002** | 서버 와이어링 | `BroadcastFunc` 캐스팅 및 `NewEngine` 시그니처 일치 | [x] |
| **TASK-003** | 서버 와이어링 | `setGameStateFunc`, `getGameStateFunc` 콜백 매핑 | [x] |
| **TASK-101** | 리팩토링 | `GameManager` 구조체 추출 (전역 상태 제거) | [x] |
| **TASK-102** | 리팩토링 | `GameService` 레이어 도입 | [x] |
| **TASK-103** | 리팩토링 | 불필요한 주석/미사용 코드 정리 | [x] |
| **TASK-104** | 리팩토링 | `RoomService` 도입 | [x] |
| **TASK-004** | 게임 로직 | 25점 즉시 승리 구현 | [x] |
| **TASK-005** | 게임 로직 | 최종 점수/종료 사유 브로드캐스트 | [x] |
| **TASK-006** | 규칙 검증 | 빈 힌트 방지 | [x] |
| **TASK-007** | 규칙 검증 | 힌트 토큰 0개 시 에러 반환 | [x] |
| **TASK-008** | 규칙 검증 | 힌트 토큰 만석 시 버리기 불가 | [x] |
| **TASK-009** | 테스트 | 통합 테스트 작성 (17개) | [x] |
| **TASK-011** | 인프라 | 레디 체크 시스템 | [x] |
| **TASK-012** | 인프라 | 재접속 처리 | [x] |
| **TASK-013** | 코드 품질 | 동시성/보안/에러 처리 강화 (코드 리뷰 기반) | [x] |
| **TASK-014** | 게임 로직 | 턴 타이머 (60초, 자동 랜덤 액션) | [x] |
| **TASK-010** | 확장 기능 | 가상 플레이어(AI) 연동 설계 | [ ] |
| **TASK-015** | 확장 기능 | 관전자 모드 | [ ] |
