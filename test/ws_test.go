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
	_ = ReadEvent(t, connA, 10*time.Second) // user.identify 응답

	connB := ConnectAndIdentify(t, wsURL, "userB_game_end", "BobEnd")
	defer connB.Close()
	_ = ReadEvent(t, connB, 10*time.Second) // user.identify 응답

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
	_ = ReadEvent(t, connB, 10*time.Second) // User B's room.join success response
	_ = ReadEvent(t, connA, 10*time.Second) // User A receives room.join notification about User B
	_ = ReadEvent(t, connB, 10*time.Second) // User B receives room.join notification about themselves

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

	time.Sleep(1 * time.Second) // 상태 업데이트 반영 대기

	// 5. User A (방장)가 게임 시작
	SendEvent(t, connA, WSEvent{
		Type: "game.start",
		Data: map[string]interface{}{"roomId": roomID},
	})
	_ = ReadEvent(t, connA, 10*time.Second) // game.sync 응답
	_ = ReadEvent(t, connA, 10*time.Second) // game.started 브로드캐스트
	_ = ReadEvent(t, connB, 10*time.Second) // game.sync 브로드캐스트
	_ = ReadEvent(t, connB, 10*time.Second) // game.started 브로드캐스트

	// 6. 게임 종료 시뮬레이션 (여기서는 미스 토큰 3개 소진으로 패배 시나리오 가정)
	// 실제 게임 로직에 따라 카드를 플레이하거나 버리는 액션을 반복하여 미스 토큰을 소진시킵니다.
	// 이 부분은 하나비 게임 규칙과 카드 처리에 따라 매우 복잡해질 수 있습니다.
	// 여기서는 간단히 미스 토큰을 소진시키는 액션을 가정합니다.
	// 주의: 이 테스트는 'engine.go' 내부 로직이 올바르게 미스 토큰을 처리하고 게임을 종료시키는 것을 전제로 합니다.
	// 실제 카드 플레이 로직 없이 미스 토큰을 감소시키는 직접적인 액션은 없습니다.
	// 따라서, 여기서는 3번의 실패 플레이를 시뮬레이션하여 미스 토큰이 모두 소진되도록 가정합니다.

	// 임시로 미스 토큰 감소를 유도하는 액션을 3회 보냅니다. (실제 게임 로직과 일치하지 않을 수 있음)
	// 이 부분은 게임 엔진에 '미스 토큰 감소' 액션이 직접 있다면 그 액션을 사용해야 합니다.
	// 현재는 'play_card'를 잘못해서 미스 토큰이 줄어드는 시나리오를 가정.
	// (TODO: 실제 한 번 플레이에 여러 미스 토큰이 줄지 않는 한, 3번의 턴을 돌면서 잘못 플레이해야 합니다.)

	// 예시: 3번의 잘못된 카드 플레이로 미스 토큰을 소진 (실제 구현에 따라 수정 필요)
	for i := 0; i < 3; i++ {
		// User A가 잘못된 카드 플레이 시도 (인덱스 0번 카드를 플레이 시도)
		// 이 카드가 불꽃놀이에 맞지 않아 미스 토큰이 1 감소한다고 가정.
		SendEvent(t, connA, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "play_card",
				"playerId":   "userA_game_end",
				"cardIndex":  float64(0), // 첫 번째 카드를 잘못 플레이했다고 가정
			},
		})
		_ = ReadEvent(t, connA, 10*time.Second) // game.action 응답
		_ = ReadEvent(t, connA, 10*time.Second) // game.sync 브로드캐스트
		_ = ReadEvent(t, connB, 10*time.Second) // game.sync 브로드캐스트

		// 턴 종료 (다음 플레이어 턴으로 넘어가기)
		SendEvent(t, connA, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "end_turn",
				"playerId":   "userA_game_end",
			},
		})
		_ = ReadEvent(t, connA, 10*time.Second) // game.action 응답
		_ = ReadEvent(t, connA, 10*time.Second) // game.sync 브로드캐스트
		_ = ReadEvent(t, connB, 10*time.Second) // game.sync 브로드캐스트

		// User B도 잘못된 카드 플레이 시도
		SendEvent(t, connB, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "play_card",
				"playerId":   "userB_game_end",
				"cardIndex":  float64(0), // 첫 번째 카드를 잘못 플레이했다고 가정
			},
		})
		_ = ReadEvent(t, connB, 10*time.Second) // game.action 응답
		_ = ReadEvent(t, connA, 10*time.Second) // game.sync 브로드캐스트
		_ = ReadEvent(t, connB, 10*time.Second) // game.sync 브로드캐스트

		// 턴 종료
		SendEvent(t, connB, WSEvent{
			Type: "game.action",
			Data: map[string]interface{}{
				"actionType": "end_turn",
				"playerId":   "userB_game_end",
			},
		})
		_ = ReadEvent(t, connB, 10*time.Second) // game.action 응답
		_ = ReadEvent(t, connA, 10*time.Second) // game.sync 브로드캐스트
		_ = ReadEvent(t, connB, 10*time.Second) // game.sync 브로드캐스트
	}

	time.Sleep(1 * time.Second) // 게임 종료 처리 대기

	// 7. game.ended 브로드캐스트 확인 (User A, User B 모두)
	gameEndedNotifA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifA.Type)
	assert.NotNil(t, gameEndedNotifA.Data.(map[string]interface{})["roomId"])

	gameEndedNotifB := ReadEvent(t, connB, 10*time.Second)
	assert.Equal(t, "game.ended", gameEndedNotifB.Type)
	assert.NotNil(t, gameEndedNotifB.Data.(map[string]interface{})["roomId"])

	// 8. 게임 상태 동기화 요청 (최종 상태 확인)
	SendEvent(t, connA, WSEvent{
		Type: "game.sync",
		Data: map[string]interface{}{"roomId": roomID},
	})
	finalGameSyncResA := ReadEvent(t, connA, 10*time.Second)
	assert.Equal(t, "game.sync", finalGameSyncResA.Type)

	// 최종 상태 검증
	finalGameStateA := finalGameSyncResA.Data.(map[string]interface{})
	assert.True(t, finalGameStateA["gameOver"].(bool), "Game should be over")
	assert.Equal(t, 0, int(finalGameStateA["missTokens"].(float64)), "Miss tokens should be 0 for game over")

	// 이 시점에서 fireworks 점수도 검증할 수 있습니다.
	// 예를 들어, 0점 패배 시나리오라면:
	// fireworksMap := finalGameStateA["fireworks"].(map[string]interface{})
	// totalScore := 0
	// for _, score := range fireworksMap {
	// 	totalScore += int(score.(float64))
	// }
	// assert.Equal(t, 0, totalScore, "Total score should be 0 if all miss tokens are used")
}
