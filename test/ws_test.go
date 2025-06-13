package test

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Ryeom/board-game/server/ws" // 기존 ws 패키지
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// startTestServer는 Echo 서버를 시작하고 WebSocket 핸들러를 등록합니다.
func startTestServer(t *testing.T) (*httptest.Server, string) {
	e := echo.New()
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

	// ConnectAndIdentify 헬퍼 함수 사용
	conn := ConnectAndIdentify(t, wsURL, "testUser1", "Tester1")
	defer conn.Close()

	// 서버가 identify에 대한 응답을 보내지 않으므로, 다음 시스템 메시지를 기다립니다.
	// ping/pong 핸들러가 설정되어 있으므로, system.ping을 보내고 pong을 받아봅니다.
	SendEvent(t, conn, WSEvent{Type: "system.ping"})
	res := ReadEvent(t, conn, 3*time.Second) // 3초 타임아웃
	assert.Equal(t, "pong", res.Type)
	assert.Equal(t, "pong", res.Data.(map[string]interface{})["message"])
}

func TestRoomCreationAndJoin(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	// 사용자 A (방장) 연결 및 식별
	connA := ConnectAndIdentify(t, wsURL, "userA", "Alice")
	defer connA.Close()

	// userA가 방 생성
	SendEvent(t, connA, WSEvent{Type: "room.create"})

	// room.create 응답 수신: room_created 및 room_list
	// room_created: {"type":"room.create","data":{"room_id":"room:userA:...","room_list":[]},"success":true}
	// room_list: {"type":"room.list","data":[{"id":"room:userA:...","host":"userA",...}],"success":true}
	roomCreatedRes := ReadEvent(t, connA, 3*time.Second)
	assert.Equal(t, "room.create", roomCreatedRes.Type)
	roomID := roomCreatedRes.Data.(map[string]interface{})["room_id"].(string) // 생성된 방 ID 추출

	roomListRes := ReadEvent(t, connA, 3*time.Second)
	assert.Equal(t, "room.list", roomListRes.Type)
	assert.Len(t, roomListRes.Data.([]interface{}), 1) // 목록에 방이 하나 있어야 함

	// 사용자 B 연결 및 식별
	connB := ConnectAndIdentify(t, wsURL, "userB", "Bob")
	defer connB.Close()

	// userB가 userA가 생성한 방에 참여
	SendEvent(t, connB, WSEvent{Type: "room.join", RoomID: roomID})

	// userB의 room.join 응답 확인
	joinResB := ReadEvent(t, connB, 3*time.Second)
	assert.Equal(t, "room.join", joinResB.Type)
	assert.Equal(t, roomID, joinResB.Data.(map[string]interface{})["id"]) // userB에게 방 정보 전송

	// userA에게 userB가 방에 참여했다는 알림 확인
	userJoinedNotifA := ReadEvent(t, connA, 3*time.Second)
	assert.Equal(t, "room.join", userJoinedNotifA.Type)
	assert.Equal(t, "userB", userJoinedNotifA.Data.(map[string]interface{})["userId"])
	assert.Equal(t, "Bob", userJoinedNotifA.Data.(map[string]interface{})["userName"])
}

func TestRoomLeave(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	// 사용자 A (방장) 연결 및 식별
	connA := ConnectAndIdentify(t, wsURL, "userA_leave", "AliceLeave")
	defer connA.Close()

	// 사용자 B 연결 및 식별
	connB := ConnectAndIdentify(t, wsURL, "userB_leave", "BobLeave")
	defer connB.Close()

	// userA가 방 생성
	SendEvent(t, connA, WSEvent{Type: "room.create"})
	roomCreatedResA := ReadEvent(t, connA, 3*time.Second) // room.create 응답
	assert.Equal(t, "room.create", roomCreatedResA.Type)
	roomID := roomCreatedResA.Data.(map[string]interface{})["room_id"].(string)

	_ = ReadEvent(t, connA, 3*time.Second) // room.list 응답

	// userB가 방에 참여
	SendEvent(t, connB, WSEvent{Type: "room.join", RoomID: roomID})
	_ = ReadEvent(t, connB, 3*time.Second) // userB의 room.join 응답
	_ = ReadEvent(t, connA, 3*time.Second) // userA에게 userB 참여 알림

	// userB가 방 나가기
	SendEvent(t, connB, WSEvent{Type: "room.leave", RoomID: roomID})

	// userB에게 room_left 응답 확인
	roomLeftResB := ReadEvent(t, connB, 3*time.Second)
	assert.Equal(t, "room_left", roomLeftResB.Type)
	assert.Equal(t, roomID, roomLeftResB.Data.(map[string]interface{})["roomId"])

	// userA에게 user_left 알림 확인
	userLeftNotifA := ReadEvent(t, connA, 3*time.Second)
	assert.Equal(t, "user.left", userLeftNotifA.Type)
	assert.Equal(t, "userB_leave", userLeftNotifA.Data.(map[string]interface{})["userId"])
}

func TestRoomUpdate(t *testing.T) {
	ts, wsURL := startTestServer(t)
	defer ts.Close()

	connA := ConnectAndIdentify(t, wsURL, "userA_update", "AliceUpdate")
	defer connA.Close()

	connB := ConnectAndIdentify(t, wsURL, "userB_update", "BobUpdate")
	defer connB.Close()

	// userA가 방 생성
	SendEvent(t, connA, WSEvent{Type: "room.create"})
	roomCreatedResA := ReadEvent(t, connA, 3*time.Second)
	roomID := roomCreatedResA.Data.(map[string]interface{})["room_id"].(string)
	_ = ReadEvent(t, connA, 3*time.Second)

	// userB 방 참여
	SendEvent(t, connB, WSEvent{Type: "room.join", RoomID: roomID})
	_ = ReadEvent(t, connB, 3*time.Second)
	_ = ReadEvent(t, connA, 3*time.Second)

	// userA (방장)가 게임 모드 업데이트
	SendEvent(t, connA, WSEvent{Type: "room.update", RoomID: roomID, Data: map[string]interface{}{"gameMode": "hanabi"}})

	// userA에게 room_updated 응답 확인
	roomUpdatedResA := ReadEvent(t, connA, 3*time.Second)
	assert.Equal(t, "room_updated", roomUpdatedResA.Type)
	assert.Equal(t, "hanabi", roomUpdatedResA.Data.(map[string]interface{})["GameMode"])

	// userB에게 room_updated 알림 확인
	roomUpdatedNotifB := ReadEvent(t, connB, 3*time.Second)
	assert.Equal(t, "room_updated", roomUpdatedNotifB.Type)
	assert.Equal(t, "hanabi", roomUpdatedNotifB.Data.(map[string]interface{})["GameMode"])
}
