package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGameFlow(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	// 1. 사용자 A (방장) 및 사용자 B 연결 및 식별
	connA := ConnectAndIdentify(t, wsURL, "userA_game", "AliceGame")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	connB := ConnectAndIdentify(t, wsURL, "userB_game", "BobGame")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second)

	// 2. User A가 방 생성
	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{"roomName": "Game Flow Test Room", "maxPlayers": 2},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	roomID := roomCreatedResA.Data.(map[string]interface{})["room_id"].(string)

	// 3. User B가 방에 참여
	SendEvent(t, connB, WSEvent{
		Type: "room.join",
		Data: map[string]interface{}{"roomId": roomID},
	})
	_ = ReadEvent(t, connB, 10*time.Second) // User B's room.join success response
	_ = ReadEvent(t, connA, 10*time.Second) // User A receives room.join notification about User B
	_ = ReadEvent(t, connB, 10*time.Second) // User B receives room.join notification about themselves

	// 4. User A, User B 상태를 "ready"로 업데이트
	SendEvent(t, connA, WSEvent{
		Type: "user.update",
		Data: map[string]interface{}{"status": "ready"},
	})
	_ = ReadEvent(t, connA, 10*time.Second)

	SendEvent(t, connB, WSEvent{
		Type: "user.update",
		Data: map[string]interface{}{"status": "ready"},
	})
	_ = ReadEvent(t, connB, 10*time.Second)

	time.Sleep(1 * time.Second)
	// 5. User A (방장)가 게임 시작
	SendEvent(t, connA, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID},
	})
	// game.start 응답 (User A)
	gameStartResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.sync", gameStartResA.Type)
	assert.NotNil(t, gameStartResA.Data, "Game state should not be nil")
	initialStateA := gameStartResA.Data.(map[string]interface{})
	initialHintTokens := int(initialStateA["hintTokens"].(float64))
	assert.Equal(t, 8, initialHintTokens, "Initial hint tokens should be 8")

	// game.state 브로드캐스트 (User A) -> game.start를 시작한 본인도 이 노티를 받지않음 -> room 내 공통 이벤트 이므로
	//gameStartedStateA := ReadEvent(t, connA, 10*time.Second)
	//assert.Equal(t, "game.start", gameStartedStateA.Type)
	//assert.Equal(t, "hanabi", gameStartedStateA.Data.(map[string]interface{})["gameMode"])

	// game.state 브로드캐스트 (User B)
	//gameStartedStateB := ReadEvent(t, connB, 10*time.Second)
	//assert.Equal(t, "game.sync", gameStartedStateB.Type)
	//assert.NotNil(t, gameStartedStateB.Data.(map[string]interface{})["fireworks"], "Game state should not be nil for User B")

	// 6. User A가 User B에게 힌트 주기 (Game Action)
	SendEvent(t, connA, WSEvent{
		Type: "game.action",
		Data: map[string]interface{}{
			"actionType": "give_hint",
			"toId":       "userB_game",
			"hintType":   "color",
			"value":      "red",
		},
	})
	// game.state 브로드캐스트 (User B) - 힌트 토큰 감소 및 카드 정보 변경 확인
	gameStartedNotiB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.started", gameStartedNotiB.Type)

	// 잘못된 요청입니다.
	gameStartedNotiA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.started", gameStartedNotiA.Type)

	// TODO : User B의 핸드에서 빨간색 카드의 ColorKnown이 true가 되었는지 확인

	// 7. User A (방장)가 게임 종료
	SendEvent(t, connA, WSEvent{
		Type: "game.end",
		Data: map[string]interface{}{"roomId": roomID},
	})

	// game.end 응답 (User A)
	gameEndResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.end", gameEndResA.Type)
	assert.Equal(t, "ended", gameEndResA.Data.(map[string]interface{})["status"])

	// game.ended 브로드캐스트 (User A)
	gameEndedNotifA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifA.Type)

	// game.ended 브로드캐스트 (User B)
	gameEndedNotifB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifB.Type)
}
