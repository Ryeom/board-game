package ws

import (
	"context"
	"fmt"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/Ryeom/board-game/room"
)

func handleEvent(user *UserSession, event SocketEvent) {
	ctx := context.Background()
	switch event.Type {
	case "create_room":
		rid := "room:" + util.GetUUID()
		host := &userSessionWrapper{user}
		r := room.CreateRoom(ctx, rid, host)
		user.Conn.WriteJSON(map[string]any{
			"type": "room_created",
			"data": map[string]string{
				"roomId": r.ID,
			},
		})
		rooms := room.ListRooms(ctx)
		user.Conn.WriteJSON(map[string]any{
			"type": "room_list",
			"data": rooms,
		})
	case "join_room":
		r, ok := room.JoinRoom(ctx, event.RoomID, user.ID)
		if !ok {
			user.Conn.WriteJSON(map[string]any{
				"type":    "error",
				"message": "room not found",
			})
			return
		}
		user.RoomID = r.ID
		user.Conn.WriteJSON(map[string]any{
			"type": "room_joined",
			"data": r,
		})
		GlobalBroadcaster.BroadcastToRoom(user.RoomID, map[string]any{
			"type": "user_joined",
			"data": map[string]string{
				"userId":   user.ID,
				"userName": user.Name,
			},
		})
	case "start_game":
		fmt.Println("🎮 start game in room:", event.RoomID)
		// TODO: room.GetRoom(event.RoomID).Engine.StartGame()
	case "get_room_list":
		rooms := room.ListRooms(ctx)
		user.Conn.WriteJSON(map[string]any{
			"type": "room_list",
			"data": rooms,
		})
	default:
		fmt.Println("⚠️ unknown event type:", event.Type)
	}
}
