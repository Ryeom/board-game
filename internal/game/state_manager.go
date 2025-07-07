package game

import (
	"context"
	"fmt"
	redisutil "github.com/Ryeom/board-game/infra/redis"
	"github.com/Ryeom/board-game/internal/domain/room"
	"github.com/Ryeom/board-game/log"
	"time"
)

func getGameStateKey(gameMode room.GameMode, roomID string) string {
	return fmt.Sprintf("game:%s:state:%s", gameMode, roomID)
}

func SaveGameState(ctx context.Context, gameMode room.GameMode, roomID string, state interface{}) error {
	key := getGameStateKey(gameMode, roomID)

	err := redisutil.SaveJSON(redisutil.RedisTargetGame, key, state, 24*time.Hour)
	if err != nil {
		log.Logger.Errorf("SaveGameState - Failed to save game state for room %s (mode %s): %v", roomID, gameMode, err)
		return fmt.Errorf("failed to save game state: %w", err)
	}
	log.Logger.Debugf("Saved game state for room %s (mode %s)", roomID, gameMode)
	return nil
}

func GetGameState(ctx context.Context, gameMode room.GameMode, roomID string, dest interface{}) error {
	key := getGameStateKey(gameMode, roomID)
	found := redisutil.GetJSON(redisutil.RedisTargetGame, key, dest)
	if !found {
		return fmt.Errorf("game state not found for room %s (mode %s)", roomID, gameMode)
	}
	log.Logger.Debugf("Loaded game state for room %s (mode %s)", roomID, gameMode)
	return nil
}

func DeleteGameState(ctx context.Context, gameMode room.GameMode, roomID string) error {
	key := getGameStateKey(gameMode, roomID)
	err := redisutil.Delete(redisutil.RedisTargetGame, key)
	if err != nil {
		log.Logger.Errorf("DeleteGameState - Failed to delete game state for room %s (mode %s): %v", roomID, gameMode, err)
		return fmt.Errorf("failed to delete game state: %w", err)
	}
	log.Logger.Debugf("Deleted game state for room %s (mode %s)", roomID, gameMode)
	return nil
}
