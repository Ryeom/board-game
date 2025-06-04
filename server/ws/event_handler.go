package ws

import (
	"context"
	"fmt"
	"github.com/Ryeom/board-game/room"
	"log"
	"time"
)

var eventHandlers = map[string]func(context.Context, *UserSession, SocketEvent){
	"room.create": HandleRoomCreate,
	"room.join":   HandleRoomJoin,
	"room.list":   HandleRoomList,
	"game.info":   HandleGameInfo,
}

func dispatchSocketEvent(user *UserSession, event SocketEvent) {
	ctx := context.Background()

	if handler, ok := eventHandlers[event.Type]; ok {
		handler(ctx, user, event)
	} else {
		log.Println("⚠️ Unknown event type:", event.Type)
	}
}
func HandleDefault(ctx context.Context, user *UserSession, event SocketEvent) {}

func HandleGameInfo(ctx context.Context, user *UserSession, event SocketEvent) {
	gameInfo := []string{}
	user.Conn.WriteJSON(map[string]any{
		"type": "room_list",
		"data": gameInfo,
	})
}
func HandleRoomCreate(ctx context.Context, user *UserSession, event SocketEvent) {
	rid := "room:" + user.ID + ":" + fmt.Sprint(time.Now().UnixNano())
	r := &room.Room{
		ID:        rid,
		Host:      user.ID,
		Players:   []string{user.ID},
		GameMode:  room.GameModeHanabi,
		CreatedAt: time.Now(),
	}
	r.Save(ctx)
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
}

// Used in eventHandlers["room.join"]
func HandleRoomJoin(ctx context.Context, user *UserSession, event SocketEvent) {
	r, ok := room.GetRoom(ctx, event.RoomID)
	if !ok {
		user.Conn.WriteJSON(map[string]any{
			"type":    "error",
			"message": "room not found",
		})
		return
	}
	user.RoomID = r.ID
	_, _ = r.Join(ctx, user.ID)
	user.Conn.WriteJSON(map[string]any{
		"type": "room_joined",
		"data": r,
	})
	//GlobalBroadcaster.BroadcastToRoom(user.RoomID, map[string]any{
	//	"type": "user_joined",
	//	"data": map[string]string{
	//		"userId":   user.ID,
	//		"userName": user.Name,
	//	},
	//})
}

// Used in eventHandlers["room.list"]
func HandleRoomList(ctx context.Context, user *UserSession, _ SocketEvent) {
	rooms := room.ListRooms(ctx)
	user.Conn.WriteJSON(map[string]any{
		"type": "room_list",
		"data": rooms,
	})
}

// Used in eventHandlers["room.start"] -- placeholder for now
func HandleRoomStart(ctx context.Context, user *UserSession, event SocketEvent) {
	fmt.Println("🎮 start game in room:", event.RoomID)
	// TODO: Start game logic
}
