package ws

import (
	"context"
	"encoding/json"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/redis/go-redis/v9"
)

func SaveUserSession(ctx context.Context, session *UserSession) error {
	key := sessionKey(session.ID)

	// Redis에는 연결 정보(Conn)는 제외하고 저장
	temp := *session
	temp.Conn = nil
	data, err := json.Marshal(temp)
	if err != nil {
		return fmt.Errorf("세션 직렬화 실패: %w", err)
	}

	pipe := redisutil.RoomClient.TxPipeline()
	pipe.Set(ctx, key, data, sessionTTL)
	pipe.SAdd(ctx, roomIndexKey(session.RoomID), session.ID)
	_, err = pipe.Exec(ctx)
	return err
}

func GetUserSession(ctx context.Context, socketID string) (*UserSession, error) {
	key := sessionKey(socketID)
	val, err := redisutil.RoomClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("Redis 조회 실패: %w", err)
	}
	var session UserSession
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, fmt.Errorf("역직렬화 실패: %w", err)
	}
	return &session, nil
}

func DeleteUserSession(ctx context.Context, socketID string) error {
	session, err := GetUserSession(ctx, socketID)
	if err != nil || session == nil {
		return err
	}

	pipe := redisutil.RoomClient.TxPipeline()
	pipe.Del(ctx, sessionKey(socketID))
	pipe.SRem(ctx, roomIndexKey(session.RoomID), socketID)
	_, err = pipe.Exec(ctx)
	return err
}

func GetSessionsByRoom(ctx context.Context, roomID string) ([]*UserSession, error) {
	ids, err := redisutil.RoomClient.SMembers(ctx, roomIndexKey(roomID)).Result()
	if err != nil {
		return nil, err
	}
	var sessions []*UserSession
	for _, id := range ids {
		s, err := GetUserSession(ctx, id)
		if err == nil && s != nil {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}
