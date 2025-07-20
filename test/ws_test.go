package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebSocketConnectionAndIdentify(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	conn := ConnectAndIdentify(t, wsURL, "testUser1", "Tester1")
	defer conn.Close()

	// 1. First ReadEvent (server's "user.identify" success response)
	resIdentify := ReadEvent(t, conn, 10*time.Second)
	assert.Equal(t, "user.identify", resIdentify.Type)
	assert.Equal(t, "testUser1", resIdentify.Data.(map[string]interface{})["userId"])

	// 2. Client sends "system.ping" request.
	SendEvent(t, conn, WSEvent{Type: "system.ping"})

	// 3. Second ReadEvent (client expects server's "pong" response)
	resPong := ReadEvent(t, conn, 10*time.Second)
	// 4. Assertions
	assert.Equal(t, "pong", resPong.Type)
	assert.Equal(t, "pong", resPong.Data.(map[string]interface{})["message"])
}

func TestRoomCreationAndJoin(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	// 사용자 A (방장) 연결 및 식별
	connA := ConnectAndIdentify(t, wsURL, "userA", "Alice")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	// 사용자 B 연결 및 식별
	connB := ConnectAndIdentify(t, wsURL, "userB", "Bob")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second)

	// userA가 방 생성
	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{
			"roomName":   "Test Room A",
			"maxPlayers": 4,
		},
	})

	// room.create 응답 수신 (room_list도 함께 포함됨)
	roomCreatedRes := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedRes.Type)

	// Data 필드 검증 강화
	assert.NotNil(t, roomCreatedRes.Data, "roomCreatedRes.Data should not be nil")
	roomCreatedDataMap, ok := roomCreatedRes.Data.(map[string]interface{})
	assert.True(t, ok, "roomCreatedRes.Data should be of type map[string]interface{}")

	// room_id 추출
	roomID := roomCreatedDataMap["room_id"].(string)

	// room_list 추출 및 검증
	roomList := roomCreatedDataMap["room_list"].([]interface{})
	assert.Len(t, roomList, 1)

	// 사용자 B가 userA가 생성한 방에 참여
	SendEvent(t, connB, WSEvent{
		Type: "room.join",
		Data: map[string]interface{}{
			"roomId": roomID,
		},
	})

	// userB의 room.join 응답 확인
	joinResB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "room.join", joinResB.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, joinResB.Data, "joinResB.Data should not be nil")
	joinResBDataMap, ok := joinResB.Data.(map[string]interface{})
	assert.True(t, ok, "joinResB.Data should be of type map[string]interface{}")

	// 서버는 Data 맵 안에 Room 객체 전체를 반환하므로, id 필드는 Data 맵 안에 있습니다.
	assert.Equal(t, roomID, joinResBDataMap["id"])

	// userA에게 userB가 방에 참여했다는 알림 확인
	userJoinedNotifA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.join", userJoinedNotifA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, userJoinedNotifA.Data, "userJoinedNotifA.Data should not be nil")
	userJoinedNotifADataMap, ok := userJoinedNotifA.Data.(map[string]interface{})
	assert.True(t, ok, "userJoinedNotifA.Data should be of type map[string]interface{}")

	assert.Equal(t, "userB", userJoinedNotifADataMap["userId"])
	assert.Equal(t, "Bob", userJoinedNotifADataMap["userName"])
}

func TestRoomLeave(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	connA := ConnectAndIdentify(t, wsURL, "userA_leave", "AliceLeave")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	connB := ConnectAndIdentify(t, wsURL, "userB_leave", "BobLeave")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{"roomName": "Test Room Leave", "maxPlayers": 4},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, roomCreatedResA.Data, "roomCreatedResA.Data should not be nil")
	roomCreatedDataMap, ok := roomCreatedResA.Data.(map[string]interface{})
	assert.True(t, ok, "roomCreatedResA.Data should be of type map[string]interface{}")
	roomID := roomCreatedDataMap["room_id"].(string)

	SendEvent(t, connB, WSEvent{
		Type: "room.join",
		Data: map[string]interface{}{"roomId": roomID},
	})
	// User B receives room.join success response
	_ = ReadEvent(t, connB, 10*time.Second)
	// User A receives room.join notification about User B
	_ = ReadEvent(t, connA, 10*time.Second)
	// User B also receives room.join notification about themselves
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connB, WSEvent{
		Type: "room.leave",
		Data: map[string]interface{}{"roomId": roomID},
	})

	// User B receives room.leave success response
	roomLeftResB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "room.leave", roomLeftResB.Type)

	assert.NotNil(t, roomLeftResB.Data, "roomLeftResB.Data should not be nil")
	roomLeftResBDataMap, ok := roomLeftResB.Data.(map[string]interface{})
	assert.True(t, ok, "roomLeftResB.Data should be of type map[string]interface{}")

	assert.Equal(t, roomID, roomLeftResBDataMap["roomId"])

	userLeftNotifA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "user.left", userLeftNotifA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, userLeftNotifA.Data, "userLeftNotifA.Data should not be nil")
	userLeftNotifADataMap, ok := userLeftNotifA.Data.(map[string]interface{})
	assert.True(t, ok, "userLeftNotifA.Data should be of type map[string]interface{}")

	assert.Equal(t, "userB_leave", userLeftNotifADataMap["userId"])
	assert.Equal(t, "BobLeave", userLeftNotifADataMap["userName"])
}

func TestRoomUpdate(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	connA := ConnectAndIdentify(t, wsURL, "userA_update", "AliceUpdate")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	connB := ConnectAndIdentify(t, wsURL, "userB_update", "BobUpdate")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{"roomName": "Original Room Name", "maxPlayers": 4},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, roomCreatedResA.Data, "roomCreatedResA.Data should not be nil")
	roomCreatedDataMap, ok := roomCreatedResA.Data.(map[string]interface{})
	assert.True(t, ok, "roomCreatedResA.Data should be of type map[string]interface{}")
	roomID := roomCreatedDataMap["room_id"].(string)

	SendEvent(t, connB, WSEvent{
		Type: "room.join",
		Data: map[string]interface{}{"roomId": roomID},
	})
	_ = ReadEvent(t, connB, 10*time.Second)
	_ = ReadEvent(t, connA, 10*time.Second)
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "room.update",
		Data: map[string]interface{}{"roomName": "Updated Room Name", "gameMode": "hanabi"},
	})

	roomUpdatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.update", roomUpdatedResA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, roomUpdatedResA.Data, "roomUpdatedResA.Data should not be nil")
	roomUpdatedResADataMap, ok := roomUpdatedResA.Data.(map[string]interface{})
	assert.True(t, ok, "roomUpdatedResA.Data should be of type map[string]interface{}")

	assert.Equal(t, "Updated Room Name", roomUpdatedResADataMap["roomName"])
	assert.Equal(t, roomID, roomUpdatedResADataMap["id"])
	assert.Equal(t, "hanabi", roomUpdatedResADataMap["gameMode"])

	roomUpdatedBroadcastA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.update", roomUpdatedBroadcastA.Type)
	roomUpdatedBroadcastADataMap, ok := roomUpdatedBroadcastA.Data.(map[string]interface{})
	assert.True(t, ok, "roomUpdatedBroadcastA.Data should be of type map[string]interface{}")
	assert.Equal(t, "Updated Room Name", roomUpdatedBroadcastADataMap["roomName"])
	assert.Equal(t, "hanabi", roomUpdatedBroadcastADataMap["gameMode"])

	roomUpdatedNotifB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "room.update", roomUpdatedNotifB.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, roomUpdatedNotifB.Data, "roomUpdatedNotifB.Data should not be nil")
	roomUpdatedNotifBDataMap, ok := roomUpdatedNotifB.Data.(map[string]interface{})
	assert.True(t, ok, "roomUpdatedNotifB.Data should be of type map[string]interface{}")

	assert.Equal(t, "Updated Room Name", roomUpdatedNotifBDataMap["roomName"])
	assert.Equal(t, "hanabi", roomUpdatedNotifBDataMap["gameMode"])
}

func TestGameStartFailureNotHost(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	connA := ConnectAndIdentify(t, wsURL, "userA_no_host", "Alice")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	connB := ConnectAndIdentify(t, wsURL, "userB_no_host", "Bob")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{"roomName": "No Host Test", "maxPlayers": 2},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, roomCreatedResA.Data, "roomCreatedResA.Data should not be nil")
	roomCreatedDataMap, ok := roomCreatedResA.Data.(map[string]interface{})
	assert.True(t, ok, "roomCreatedResA.Data should be of type map[string]interface{}")
	roomID := roomCreatedDataMap["room_id"].(string)

	SendEvent(t, connB, WSEvent{
		Type: "room.join",
		Data: map[string]interface{}{"roomId": roomID},
	})
	_ = ReadEvent(t, connB, 10*time.Second)
	_ = ReadEvent(t, connA, 10*time.Second)
	_ = ReadEvent(t, connB, 10*time.Second)

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

	SendEvent(t, connB, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID},
	})
	errorResB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "error", errorResB.Type)
	assert.Equal(t, "ERROR_ROOM_NOT_HOST", errorResB.ErrorCode)
}

func TestGameStartFailureNotEnoughPlayers(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	connA := ConnectAndIdentify(t, wsURL, "userA_one_player", "Alice")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{"roomName": "Not Enough Players Test", "maxPlayers": 2},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, roomCreatedResA.Data, "roomCreatedResA.Data should not be nil")
	roomCreatedDataMap, ok := roomCreatedResA.Data.(map[string]interface{})
	assert.True(t, ok, "roomCreatedResA.Data should be of type map[string]interface{}")
	roomID := roomCreatedDataMap["room_id"].(string)

	SendEvent(t, connA, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID}, // Added roomId to data
	})
	errorResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "error", errorResA.Type)
	assert.Equal(t, "ERROR_GAME_NOT_ENOUGH_PLAYERS", errorResA.ErrorCode)
}

func TestGameStartFailureNotAllReady(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	connA := ConnectAndIdentify(t, wsURL, "userA_not_ready", "Alice")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	connB := ConnectAndIdentify(t, wsURL, "userB_not_ready", "Bob")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{"roomName": "Not All Ready Test", "maxPlayers": 2},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	// Data 필드 검증 강화
	assert.NotNil(t, roomCreatedResA.Data, "roomCreatedResA.Data should not be nil")
	roomCreatedDataMap, ok := roomCreatedResA.Data.(map[string]interface{})
	assert.True(t, ok, "roomCreatedResA.Data should be of type map[string]interface{}")
	roomID := roomCreatedDataMap["room_id"].(string)

	SendEvent(t, connB, WSEvent{
		Type: "room.join",
		Data: map[string]interface{}{"roomId": roomID},
	})
	_ = ReadEvent(t, connB, 10*time.Second)
	_ = ReadEvent(t, connA, 10*time.Second)
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "user.update",
		Data: map[string]interface{}{"status": "ready"},
	})
	_ = ReadEvent(t, connA, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID},
	})
	errorResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "error", errorResA.Type)
	assert.Equal(t, "ERROR_GAME_NOT_ALL_PLAYERS_READY", errorResA.ErrorCode)
}

func TestGameInfoFetch(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	// 1. 사용자 연결 및 식별
	connA := ConnectAndIdentify(t, wsURL, "userA_info", "AliceInfo")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second) // user.identify 응답

	// 2. game.info 이벤트 전송 (하나비 게임 모드 요청)
	SendEvent(t, connA, WSEvent{
		Type: "game.info",
		Data: map[string]interface{}{"gameMode": "hanabi"},
	})

	// 3. 서버 응답 수신
	gameInfoRes := ReadEvent(t, connA, 10*time.Second)

	// 4. 응답 검증
	assert.Equal(t, "game.info", gameInfoRes.Type)
	assert.True(t, gameInfoRes.Success)
	assert.Equal(t, 200, int(gameInfoRes.Code.(float64)))
	assert.Equal(t, "SUCCESS_SYSTEM_OK", gameInfoRes.ErrorCode)

	assert.NotNil(t, gameInfoRes.Data, "Game info data should not be nil")
	dataMap, ok := gameInfoRes.Data.(map[string]interface{})
	assert.True(t, ok, "Data should be a map[string]interface{}")

	assert.Equal(t, "hanabi", dataMap["gameMode"])

	infoMap, ok := dataMap["info"].(map[string]interface{})
	assert.True(t, ok, "Info should be a map[string]interface{}")
	assert.Equal(t, "Hanabi", infoMap["name"])
	assert.Contains(t, infoMap["description"].(string), "협력 카드 게임")

	rulesSummary, ok := infoMap["rulesSummary"].([]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(rulesSummary), 0)
}
func TestHanabiGameCompletionAndScore(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	// 1. 사용자 A (방장) 및 사용자 B 연결 및 식별
	connA := ConnectAndIdentify(t, wsURL, "userA_game_end", "AliceEnd")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second) // user.identify for A

	connB := ConnectAndIdentify(t, wsURL, "userB_game_end", "BobEnd")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second) // user.identify for B

	// 2. User A가 하나비 방 생성 (2인 플레이)
	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{
			"roomName":   "Hanabi Completion Test Room",
			"maxPlayers": 2,
			"gameMode":   "hanabi", // 하나비 모드 명시
		},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	roomID := roomCreatedResA.Data.(map[string]interface{})["room_id"].(string)

	// 3. User B가 방에 참여
	SendEvent(t, connB, WSEvent{
		Type: "room.join",
		Data: map[string]interface{}{
			"roomId": roomID,
		},
	})
	_ = ReadEvent(t, connB, 10*time.Second)
	_ = ReadEvent(t, connA, 10*time.Second)
	_ = ReadEvent(t, connB, 10*time.Second)

	// 4. 모든 플레이어 상태를 "ready"로 업데이트
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

	// 5. User A (방장)가 게임 시작
	SendEvent(t, connA, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID},
	})
	gameStartedBroadcastA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.started", gameStartedBroadcastA.Type, "User A should receive 'game.started' broadcast")
	assert.NotNil(t, gameStartedBroadcastA.Data.(map[string]interface{})["state"], "Game state should not be nil for A")

	gameStartedBroadcastB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.started", gameStartedBroadcastB.Type, "User B should receive 'game.started' broadcast")
	assert.NotNil(t, gameStartedBroadcastB.Data.(map[string]interface{})["state"], "Game state should not be nil for B")

	// 6. 게임 종료 시뮬레이션 (미스 토큰 3개 소진으로 패배 시나리오 가정)
	for i := 0; i < 3; i++ {
		SendEvent(t, connA, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "play_card",
				"cardIndex":  float64(0),
			},
		})
		// A gets action success response
		actionResA := ReadEvent(t, connA, 10*time.Second)
		assert.Equal(t, "game.action", actionResA.Type)
		assert.True(t, actionResA.Success)
		// Both A and B get state update broadcast
		_ = ReadEvent(t, connA, 10*time.Second) // game.state_update A
		_ = ReadEvent(t, connB, 10*time.Second) // game.state_update B

		SendEvent(t, connA, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "end_turn",
			},
		})
		// A gets action success response
		_ = ReadEvent(t, connA, 10*time.Second)
		_ = ReadEvent(t, connA, 10*time.Second) // game.state_update A
		_ = ReadEvent(t, connB, 10*time.Second) // game.state_update B

		SendEvent(t, connB, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "play_card",
				"cardIndex":  float64(0), // Play the first card
			},
		})
		// B gets action success response
		_ = ReadEvent(t, connB, 10*time.Second)
		_ = ReadEvent(t, connA, 10*time.Second) // game.state_update A
		_ = ReadEvent(t, connB, 10*time.Second) // game.state_update B

		SendEvent(t, connB, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "end_turn",
			},
		})

		_ = ReadEvent(t, connB, 10*time.Second)
		_ = ReadEvent(t, connA, 10*time.Second) // game.state_update A
		_ = ReadEvent(t, connB, 10*time.Second) // game.state_update B
	}

	time.Sleep(1 * time.Second) // Give server time to register game over

	// 7. game.ended broadcast confirmation (User A, User B both)
	gameEndedNotifA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifA.Type, "User A should receive 'game.ended' broadcast")
	assert.NotNil(t, gameEndedNotifA.Data.(map[string]interface{})["roomId"])

	gameEndedNotifB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifB.Type, "User B should receive 'game.ended' broadcast")
	assert.NotNil(t, gameEndedNotifB.Data.(map[string]interface{})["roomId"])

	// 8. Game sync request (to verify final state)
	SendEvent(t, connA, WSEvent{
		Type: "game.sync",
		Data: map[string]interface{}{"roomId": roomID},
	})
	finalGameSyncResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.sync", finalGameSyncResA.Type)
	assert.True(t, finalGameSyncResA.Success)
	assert.Equal(t, "SUCCESS_GAME_SYNC", finalGameSyncResA.ErrorCode)

	finalGameStateA := finalGameSyncResA.Data.(map[string]interface{})["gameState"].(map[string]interface{})
	assert.True(t, finalGameStateA["gameOver"].(bool), "Game should be over")
	assert.Equal(t, float64(0), finalGameStateA["missTokens"], "Miss tokens should be 0 for game over")

	fireworksMap := finalGameStateA["fireworks"].(map[string]interface{})
	totalScore := 0
	for _, score := range fireworksMap {
		totalScore += int(score.(float64))
	}
	assert.Equal(t, 0, totalScore, "Total score should be 0 if game ends by all miss tokens used (assuming no successful plays)")
}
