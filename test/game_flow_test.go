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
	_ = ReadEvent(t, connA, 10*time.Second) // user.identify for A

	connB := ConnectAndIdentify(t, wsURL, "userB_game", "BobGame")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second) // user.identify for B

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
	_ = ReadEvent(t, connA, 10*time.Second) // user.update for A

	SendEvent(t, connB, WSEvent{
		Type: "user.update",
		Data: map[string]interface{}{"status": "ready"},
	})
	_ = ReadEvent(t, connB, 10*time.Second) // user.update for B

	time.Sleep(1 * time.Second) // Allow server to process updates

	// 5. User A (방장)가 게임 시작
	SendEvent(t, connA, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID},
	})

	// EXPECTATION: HandleGameStart triggers a broadcast of type "game.started" to all players.
	// This assumes the server-side code is fixed to send "game.started" as the type.
	gameStartedBroadcastA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.started", gameStartedBroadcastA.Type, "User A should receive 'game.started' broadcast")
	assert.NotNil(t, gameStartedBroadcastA.Data.(map[string]interface{})["state"], "Game state should not be nil for A")
	initialStateA := gameStartedBroadcastA.Data.(map[string]interface{})["state"].(map[string]interface{})
	initialHintTokens := int(initialStateA["hintTokens"].(float64))
	assert.Equal(t, 8, initialHintTokens, "Initial hint tokens should be 8")

	gameStartedBroadcastB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.started", gameStartedBroadcastB.Type, "User B should receive 'game.started' broadcast")
	assert.NotNil(t, gameStartedBroadcastB.Data.(map[string]interface{})["state"], "Game state should not be nil for B")

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
	// EXPECTATION: HandleGameAction sends a direct success response to the initiator, then a state update broadcast to all.
	actionResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.action", actionResA.Type, "User A should receive 'game.action' success response")
	assert.True(t, actionResA.Success)
	assert.Equal(t, "SUCCESS_GAME_ACTION", actionResA.ErrorCode)

	// EXPECTATION: Both players receive a state update broadcast.
	// This assumes the server-side code is fixed to send "game.state_update" as the type.
	stateUpdateBroadcastA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.state_update", stateUpdateBroadcastA.Type, "User A should receive 'game.state_update' broadcast")
	updatedStateA := stateUpdateBroadcastA.Data.(map[string]interface{})["state"].(map[string]interface{})
	assert.Equal(t, initialHintTokens-1, int(updatedStateA["hintTokens"].(float64)), "Hint tokens should decrease by 1")

	stateUpdateBroadcastB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.state_update", stateUpdateBroadcastB.Type, "User B should receive 'game.state_update' broadcast")
	updatedStateB := stateUpdateBroadcastB.Data.(map[string]interface{})["state"].(map[string]interface{})
	assert.Equal(t, initialHintTokens-1, int(updatedStateB["hintTokens"].(float64)), "Hint tokens should decrease by 1")

	// 7. User A (방장)가 게임 종료
	SendEvent(t, connA, WSEvent{
		Type: "game.end",
		Data: map[string]interface{}{"roomId": roomID},
	})

	// EXPECTATION: HandleGameEnd sends a direct success response to the initiator, then "game.ended" broadcast to all.
	gameEndResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.end", gameEndResA.Type, "User A should receive 'game.end' success response")
	assert.True(t, gameEndResA.Success)
	assert.Equal(t, "SUCCESS_GAME_END", gameEndResA.ErrorCode)

	gameEndedNotifA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifA.Type, "User A should receive 'game.ended' broadcast")
	assert.NotNil(t, gameEndedNotifA.Data.(map[string]interface{})["roomId"])

	gameEndedNotifB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifB.Type, "User B should receive 'game.ended' broadcast")
	assert.NotNil(t, gameEndedNotifB.Data.(map[string]interface{})["roomId"])
}
