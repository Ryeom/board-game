package ai

import (
	"fmt"
	"strings"
)

const AIPlayerPrefix = "ai_"

// IsAIPlayer AI 플레이어인지 판별한다.
func IsAIPlayer(playerID string) bool {
	return strings.HasPrefix(playerID, AIPlayerPrefix)
}

// GenerateAIPlayerID AI 플레이어 ID를 생성한다. (예: "ai_1", "ai_2")
func GenerateAIPlayerID(index int) string {
	return fmt.Sprintf("%s%d", AIPlayerPrefix, index)
}

// GenerateAIPlayerName AI 플레이어 표시 이름을 생성한다. (예: "Bot 1", "Bot 2")
func GenerateAIPlayerName(index int) string {
	return fmt.Sprintf("Bot %d", index)
}
