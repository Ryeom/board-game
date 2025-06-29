package ws

import (
	"context"
	chat "github.com/Ryeom/board-game/internal/domain/chat"
	apperr "github.com/Ryeom/board-game/internal/errors"
	"github.com/Ryeom/board-game/internal/user"
	"github.com/Ryeom/board-game/log"
	"time"
)

// HandleChatSend 채팅 메시지 전송
func HandleChatSend(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, apperr.BadRequest(apperr.ErrorCodeChatNotInRoom, nil))
		return
	}

	messageContent, ok := event.Data["message"].(string)
	if !ok || messageContent == "" {
		sendError(u, apperr.BadRequest(apperr.ErrorCodeChatEmptyMessage, nil))
		return
	}

	chatRecord := chat.ChatRecord{
		SenderID:   u.ID,
		SenderName: u.Name,
		Message:    messageContent,
		Timestamp:  time.Now(),
	}

	if err := chat.SaveChatMessage(ctx, u.RoomID, &chatRecord); err != nil {
		log.Logger.Errorf("HandleChatSend - Failed to save chat message via chat service for room %s: %v", u.RoomID, err)
		sendError(u, apperr.InternalServerError(apperr.ErrorCodeChatSendFailed, err))
		return
	}

	GlobalBroadcaster.BroadcastToRoom(u.RoomID, map[string]any{
		"type": "chat.message",
		"data": chatRecord,
	})

	sendResult(u, event.Type, map[string]string{"status": "sent"}, "메시지 전송 성공")
}

// HandleChatHistory 채팅 내역 조회
func HandleChatHistory(ctx context.Context, u *user.Session, event SocketEvent) {
	if u.RoomID == "" {
		sendError(u, apperr.BadRequest(apperr.ErrorCodeChatNotInRoom, nil))
		return
	}

	chatRecords, err := chat.GetChatHistory(ctx, u.RoomID)
	if err != nil {
		log.Logger.Errorf("HandleChatHistory - Failed to retrieve chat history via chat service for room %s: %v", u.RoomID, err)
		sendError(u, apperr.InternalServerError(apperr.ErrorCodeChatHistoryFetchFailed, err))
		return
	}

	sendResult(u, event.Type, map[string]any{
		"roomId":  u.RoomID,
		"history": chatRecords,
	}, "채팅 내역 조회 성공")
}
