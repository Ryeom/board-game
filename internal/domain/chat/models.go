package chat

import "time"

type ChatRecord struct {
	SenderID   string    `json:"senderId"`   // 메시지 보낸 사용자 ID
	SenderName string    `json:"senderName"` // 메시지 보낸 사용자 닉네임
	Message    string    `json:"message"`    // 채팅 내용
	Timestamp  time.Time `json:"timestamp"`  // 메시지 전송 시간
}
