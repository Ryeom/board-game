package http

import (
	"github.com/Ryeom/board-game/internal/errors"
	"net/http"
) // http 패키지 임포트 필요

type HttpResult struct {
	Code    int         `json:"code"`           // HTTP 상태 코드 (예: 200, 400, 500)
	Message string      `json:"message"`        // 클라이언트에게 보여줄 메시지
	Data    interface{} `json:"data,omitempty"` // 실제 응답 데이터 (선택 사항)
	Status  string      `json:"status"`         // 응답 상태 (예: "success", "error")
}

func Success(data any, message string) HttpResult { // message 인자 추가 (선택 사항)
	return HttpResult{
		Code:    http.StatusOK,
		Message: message, // 메시지 추가
		Data:    data,
		Status:  "success",
	}
}

func NewErrorResponse(appErr *errors.AppError) HttpResult {
	return HttpResult{
		Code:    appErr.Status,
		Message: appErr.Message,
		Data:    nil, // 에러 응답에는 데이터 없음
		Status:  "error",
	}
}
