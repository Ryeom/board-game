package ws

import "time"

type RoomCreateRequest struct {
	RoomName   string `json:"roomName"`
	Password   string `json:"password,omitempty"`
	MaxPlayers int    `json:"maxPlayers"`
}

type RoomJoinRequest struct {
	RoomID   string `json:"roomId"`
	Password string `json:"password,omitempty"`
}

type RoomKickRequest struct {
	UserID string `json:"userId"`
}

type RoomCreateResponse struct {
	RoomID     string        `json:"roomId"`
	RoomName   string        `json:"roomName"`
	MaxPlayers int           `json:"maxPlayers"`
	RoomList   []RoomSummary `json:"roomList"`
}

type RoomLeaveResponse struct {
	RoomID  string `json:"roomId"`
	NewHost string `json:"newHost"`
	Deleted bool   `json:"deleted"`
}

type RoomListResponse struct {
	Rooms []RoomSummary `json:"rooms"`
}

type RoomSummary struct {
	ID          string    `json:"id"`
	RoomName    string    `json:"roomName"`
	Host        string    `json:"host"`
	PlayerCount int       `json:"playerCount"`
	MaxPlayers  int       `json:"maxPlayers"`
	GameMode    string    `json:"gameMode"`
	HasPassword bool      `json:"hasPassword"`
	CreatedAt   time.Time `json:"createdAt"`
}
