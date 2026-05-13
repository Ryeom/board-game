package ws

type UserIdentifyRequest struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
}

type UserUpdateRequest struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

type UserStatusRequest struct {
	UserID string `json:"userId"`
}
