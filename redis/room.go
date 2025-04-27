package redisutil

import (
	"context"
	"fmt"
)

func DeleteRoomFromRedis(ctx context.Context, roomID string) error {
	key := fmt.Sprintf("room:%s", roomID)

	return nil
}
