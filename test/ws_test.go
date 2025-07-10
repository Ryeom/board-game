package test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	redisutil "github.com/Ryeom/board-game/infra/redis"
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/server"
	"github.com/Ryeom/board-game/server/ws" // Ensure ws is imported
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Redis 데이터를 정리
func cleanRedis(t *testing.T) {
	ctx := context.Background()

	// 각 Redis 타겟별로 정리할 키 패턴 정의
	cleanupMap := map[string][]string{
		redisutil.RedisTargetRoom: {"room:*"},                                               // 방 데이터
		redisutil.RedisTargetUser: {"user:session:*", "jwt:blacklist:*", "room_sessions:*"}, // 사용자 세션 및 JWT 블랙리스트
		redisutil.RedisTargetGame: {"game:*"},                                               // 게임 상태 데이터
	}

	for target, patterns := range cleanupMap {
		rdbClient := redisutil.Client[target]
		if rdbClient == nil {
			t.Fatalf("Redis client for target %s is not initialized.", target)
		}

		for _, pattern := range patterns {
			keys, err := rdbClient.Keys(ctx, pattern).Result()
			if err != nil {
				t.Fatalf("Failed to get keys for target %s, pattern %s: %v", target, pattern, err)
			}
			if len(keys) > 0 {
				_, err := rdbClient.Del(ctx, keys...).Result()
				if err != nil {
					t.Fatalf("Failed to delete keys for target %s, pattern %s: %v", target, pattern, err)
				}
				t.Logf("Cleaned %d keys for target %s, pattern %s", len(keys), target, pattern)
			} else {
				t.Logf("No keys found for target %s, pattern %s", target, pattern)
			}
		}
	}
	t.Logf("Finished Redis cleanup across all relevant targets.")
}

// startTestServer Echo 서버를 시작 및 WebSocket 핸들러를 등록
func startTestServer(t *testing.T) (*httptest.Server, string) {
	oldArgs := os.Args
	os.Args = []string{oldArgs[0], "local"}
	defer func() { os.Args = oldArgs }()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Failed to get current file path for test setup")
	}
	testDir := filepath.Dir(filename)
	projectRoot := filepath.Dir(testDir)

	viper.AddConfigPath(projectRoot)
	viper.SetConfigName("settings")
	viper.SetConfigType("toml")

	err := l.InitializeApplicationLog()
	if err != nil {
		t.Fatalf("Failed to initialize application log for tests: %v", err)
	}

	err = server.SetEnv()
	if err != nil {
		t.Fatalf("Failed to set environment for tests: %v", err)
	}

	err = server.SetConfig()
	if err != nil {
		t.Fatalf("Failed to load server configuration for tests: %v", err)
	}

	e := echo.New()
	server.Initialize(e)

	// Redis 클린업 호출
	cleanRedis(t)

	e.GET("/ws", func(c echo.Context) error {
		return ws.Websocket(c)
	})

	ts := httptest.NewServer(e)
	u, err := url.Parse(ts.URL)
	assert.NoError(t, err)
	u.Scheme = "ws"
	u.Path = "/ws"

	return ts, u.String()
}

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
	assert.Equal(t, "game.start", gameStartResA.Type)
	assert.NotNil(t, gameStartResA.Data.(map[string]interface{})["gameState"], "Game state should not be nil")
	initialStateA := gameStartResA.Data.(map[string]interface{})["gameState"].(map[string]interface{})
	initialHintTokens := int(initialStateA["hintTokens"].(float64))
	assert.Equal(t, 8, initialHintTokens, "Initial hint tokens should be 8")

	// game.state 브로드캐스트 (User A)
	gameStartedStateA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.state", gameStartedStateA.Type)
	assert.NotNil(t, gameStartedStateA.Data.(map[string]interface{})["fireworks"], "Game state should not be nil for User B")

	// game.state 브로드캐스트 (User B)
	gameStartedStateB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.state", gameStartedStateB.Type)
	assert.NotNil(t, gameStartedStateB.Data.(map[string]interface{})["fireworks"], "Game state should not be nil for User B")

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

	// game.action 응답 (User A)
	gameActionResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.action", gameActionResA.Type)
	assert.Equal(t, "action processed", gameActionResA.Data.(map[string]interface{})["status"])

	// game.state 브로드캐스트 (User A) - 힌트 토큰 감소 확인
	gameStateUpdateA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.state", gameStateUpdateA.Type)
	updatedStateA := gameStateUpdateA.Data.(map[string]interface{})
	updatedHintTokensA := int(updatedStateA["hintTokens"].(float64))
	assert.Equal(t, initialHintTokens-1, updatedHintTokensA, "Hint tokens should decrease by 1 for User A")

	// game.state 브로드캐스트 (User B) - 힌트 토큰 감소 및 카드 정보 변경 확인
	gameStateUpdateB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.state", gameStateUpdateB.Type)
	updatedStateB := gameStateUpdateB.Data.(map[string]interface{})
	updatedHintTokensB := int(updatedStateB["hintTokens"].(float64))
	assert.Equal(t, initialHintTokens-1, updatedHintTokensB, "Hint tokens should decrease by 1 for User B")

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

func TestGameEndFailureNotHost(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	connA := ConnectAndIdentify(t, wsURL, "userA_end_no_host", "Alice")
	defer connA.Close()
	_ = ReadEvent(t, connA, 10*time.Second)

	connB := ConnectAndIdentify(t, wsURL, "userB_end_no_host", "Bob")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second)

	SendEvent(t, connA, WSEvent{
		Type: "room.create",
		Data: map[string]interface{}{"roomName": "End No Host Test", "maxPlayers": 2},
	})
	roomCreatedResA := ReadEvent(t, connA, 10*time.Second)
	roomID := roomCreatedResA.Data.(map[string]interface{})["room_id"].(string)

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

	SendEvent(t, connA, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID},
	})
	_ = ReadEvent(t, connA, 10*time.Second) // game.start response
	_ = ReadEvent(t, connA, 10*time.Second) // game.started broadcast
	_ = ReadEvent(t, connB, 10*time.Second) // game.state broadcast
	_ = ReadEvent(t, connB, 10*time.Second) // game.started broadcast

	SendEvent(t, connB, WSEvent{
		Type: "game.end",
		Data: map[string]interface{}{"roomId": roomID},
	})
	errorResB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "error", errorResB.Type)
	assert.Equal(t, "ERROR_ROOM_NOT_HOST", errorResB.ErrorCode)
}
