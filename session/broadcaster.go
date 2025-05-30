package session

var GlobalBroadcaster Broadcaster = &LocalBroadcaster{}

type Broadcaster interface {
	BroadcastToRoom(roomID string, payload any)
}

func InitializeBroadcaster() {
	//client := redis.NewClient(&redis.Options{
	//	Addr: "localhost:6379",
	//	DB:   3,
	//})
	//GlobalBroadcaster = NewRedisBroadcaster(client)
}
