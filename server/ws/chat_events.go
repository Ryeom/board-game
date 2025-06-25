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
	// 사용자가 방에 속해 있는지 확인
	if u.RoomID == "" {
		sendError(u, apperr.BadRequest(apperr.ErrorCodeChatNotInRoom, nil))
		return
	}

	// 메시지 내용 추출
	messageContent, ok := event.Data["message"].(string)
	if !ok || messageContent == "" {
		sendError(u, apperr.BadRequest(apperr.ErrorCodeChatEmptyMessage, nil))
		return
	}

	// ChatRecord 생성
	chatRecord := chat.ChatRecord{
		SenderID:   u.ID,
		SenderName: u.Name,
		Message:    messageContent,
		Timestamp:  time.Now(),
	}

	// internal/domain/chat 서비스 함수 호출하여 메시지 저장
	if err := chat.SaveChatMessage(ctx, u.RoomID, &chatRecord); err != nil {
		log.Logger.Errorf("HandleChatSend - Failed to save chat message via chat service for room %s: %v", u.RoomID, err)
		sendError(u, apperr.InternalServerError(apperr.ErrorCodeChatSendFailed, err))
		return
	}

	// 해당 방의 모든 플레이어에게 채팅 메시지 브로드캐스트
	// GlobalBroadcaster는 server/ws/broadcaster.go 에 정의
	GlobalBroadcaster.BroadcastToRoom(u.RoomID, map[string]any{
		"type": "chat.message",
		"data": chatRecord, // 구조화된 채팅 메시지 전송
	})

	// 메시지 전송 성공 응답 (보낸 본인에게만)
	sendResult(u, event.Type, map[string]string{"status": "sent"}, "메시지 전송 성공")
}

// HandleChatHistory 채팅 내역 조회
func HandleChatHistory(ctx context.Context, u *user.Session, event SocketEvent) {
	// 사용자가 방에 속해 있는지 확인
	if u.RoomID == "" {
		sendError(u, apperr.BadRequest(apperr.ErrorCodeChatNotInRoom, nil))
		return
	}

	// internal/domain/chat 서비스 함수 호출하여 채팅 내역 조회
	chatRecords, err := chat.GetChatHistory(ctx, u.RoomID)
	if err != nil {
		log.Logger.Errorf("HandleChatHistory - Failed to retrieve chat history via chat service for room %s: %v", u.RoomID, err)
		sendError(u, apperr.InternalServerError(apperr.ErrorCodeChatHistoryFetchFailed, err))
		return
	}

	// 조회된 채팅 내역 응답 (요청한 본인에게만)
	sendResult(u, event.Type, map[string]any{
		"roomId":  u.RoomID,
		"history": chatRecords,
	}, "채팅 내역 조회 성공")
}
