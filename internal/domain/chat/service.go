package chat

import (
	"context" // 컨텍스트는 필요 없지만, redisutil 함수가 컨텍스트를 받으므로 여기서도 인자로 받음
	"encoding/json"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis" // redisutil 임포트
	"github.com/Ryeom/board-game/log"                   // log 임포트
)

const MaxChatHistory = 50 // 방당 저장할 최대 채팅 메시지 수

// GetChatKey는 특정 방의 채팅 내역을 저장할 Redis 키를 반환합니다.
func GetChatKey(roomID string) string {
	return fmt.Sprintf("chat:room:%s", roomID)
}

// SaveChatMessage는 채팅 메시지를 Redis LIST에 저장하고, 리스트 크기를 제한합니다.
func SaveChatMessage(ctx context.Context, roomID string, record *ChatRecord) error {
	chatKey := GetChatKey(roomID)

	// ChatRecord를 JSON 문자열로 직렬화
	recordJSON, err := json.Marshal(record)
	if err != nil {
		log.Logger.Errorf("chat.Service - Failed to marshal chat record for room %s: %v", roomID, err)
		return fmt.Errorf("failed to marshal chat record: %w", err)
	}

	// Redis LIST의 오른쪽에 메시지 추가
	if err := redisutil.RPushList(redisutil.RedisTargetPubSub, chatKey, string(recordJSON)); err != nil { // RedisTargetPubSub 사용
		log.Logger.Errorf("chat.Service - Failed to save chat message to Redis LIST for room %s: %v", roomID, err)
		return fmt.Errorf("failed to save chat message: %w", err)
	}

	// LIST 크기 제한 (오래된 메시지 삭제)
	if err := redisutil.LTrimList(redisutil.RedisTargetPubSub, chatKey, -MaxChatHistory, -1); err != nil { // RedisTargetPubSub 사용
		log.Logger.Warningf("chat.Service - Failed to trim chat history for room %s: %v", roomID, err)
		// 이 에러는 메시지 저장 자체를 실패로 보지 않으므로 에러를 반환하지 않고 로깅만
	}

	return nil
}

// GetChatHistory는 특정 방의 최근 채팅 내역을 Redis LIST에서 조회합니다.
func GetChatHistory(ctx context.Context, roomID string) ([]*ChatRecord, error) {
	chatKey := GetChatKey(roomID)

	// Redis LIST에서 채팅 내역 조회 (최근 MaxChatHistory 개)
	chatHistoryJSONs, err := redisutil.LRangeList(redisutil.RedisTargetPubSub, chatKey, -MaxChatHistory, -1) // RedisTargetPubSub 사용
	if err != nil {
		log.Logger.Errorf("chat.Service - Failed to retrieve chat history for room %s: %v", roomID, err)
		return nil, fmt.Errorf("failed to retrieve chat history: %w", err)
	}

	var chatRecords []*ChatRecord
	for _, jsonStr := range chatHistoryJSONs {
		var record ChatRecord
		if err := json.Unmarshal([]byte(jsonStr), &record); err != nil {
			log.Logger.Errorf("chat.Service - Failed to unmarshal chat record from Redis: %v, JSON: %s", err, jsonStr)
			continue // 유효하지 않은 레코드는 건너뛰고 다음으로 진행
		}
		chatRecords = append(chatRecords, &record)
	}

	return chatRecords, nil
}
