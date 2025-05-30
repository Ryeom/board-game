package session

import "encoding/json"

type LocalBroadcaster struct{}

func (b *LocalBroadcaster) BroadcastToRoom(roomID string, payload any) {
	msg, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, sess := range sessions {
		if sess.RoomID == roomID {
			_ = sess.Connection.WriteMessage(1, msg) // 1 = TextMessage
		}
	}
}
