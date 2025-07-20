package test

import (
	"context"
	"encoding/json"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	l "github.com/Ryeom/board-game/log"
	"github.com/Ryeom/board-game/server"
	"github.com/Ryeom/board-game/server/ws"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
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

func ReadEventsUntilCount(t *testing.T, conn *websocket.Conn, count int, timeout time.Duration) []WSEvent {
	events := make([]WSEvent, 0, count)
	deadline := time.Now().Add(timeout)

	for len(events) < count {
		// 남은 시간 계산하여 SetReadDeadline 설정
		remainingTime := time.Until(deadline)
		if remainingTime <= 0 {
			t.Errorf("Timeout reading messages. Only received %d of %d expected.", len(events), count)
			break // 타임아웃
		}
		err := conn.SetReadDeadline(time.Now().Add(remainingTime))
		if err != nil {
			t.Fatalf("Failed to set read deadline: %v", err)
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				t.Errorf("Read timeout before receiving all messages. Received %d of %d expected.", len(events), count)
				break
			}
			t.Fatalf("Failed to read message: %v", err) // 그 외 에러는 치명적
		}

		var event WSEvent
		err = json.Unmarshal(msg, &event)
		assert.NoError(t, err, "Failed to unmarshal message: %s", string(msg))
		events = append(events, event)
	}
	return events
}

// FindEventByType 주어진 이벤트 목록에서 특정 타입의 이벤트를 찾아 반환
func FindEventByType(events []WSEvent, eventType string) (WSEvent, bool) {
	for _, event := range events {
		if event.Type == eventType {
			return event, true
		}
	}
	return WSEvent{}, false
}
