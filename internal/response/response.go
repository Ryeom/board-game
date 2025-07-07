package response

import (
	"time"
)

type ErrorDetail struct {
	Field   string `json:"field"`   // 오류가 발생한 필드 이름
	Message string `json:"message"` // 해당 필드의 오류 메시지
}

type HttpError struct {
	Code             string        `json:"code"`                       // 내부 오류 코드
	Message          string        `json:"message"`                    // 사용자에게 보여줄 오류 메시지
	DeveloperMessage string        `json:"developerMessage,omitempty"` // 개발자를 위한 상세 메시지 (선택 사항)
	Details          []ErrorDetail `json:"details,omitempty"`          // 특정 필드 오류 목록
	TraceID          string        `json:"traceId,omitempty"`          // 문제 추적용 ID
	Severity         string        `json:"severity,omitempty"`
	Action           string        `json:"action,omitempty"`
	HttpStatusCode   int           `json:"httpStatusCode,omitempty"`
}
type HttpResult struct {
	Message string      `json:"message"`        // 클라이언트에게 보여줄 메시지
	Data    interface{} `json:"data,omitempty"` // 실제 응답 데이터
	Status  string      `json:"status"`         // 응답 상태 (예: "success", "error")
	Code    string      `json:"code,omitempty"` // 내부오류코드

	Timestamp time.Time  `json:"timestamp,omitempty"`
	TraceID   string     `json:"traceId,omitempty"`
	Error     *HttpError `json:"error,omitempty"` // 오류 발생 시 오류 상세 정보
}

func Success(resultCode string, data any, lang string) HttpResult {
	codeDefine, ok := GetDefineCode(resultCode, lang)
	if !ok {
		return HttpResult{
			Message:   "The result code is not found.",
			Status:    "success",
			Data:      data,
			Code:      resultCode,
			Timestamp: time.Now(),
		}
	}

	return HttpResult{
		Message:   codeDefine.Message,
		Data:      data,
		Status:    "success",
		Code:      resultCode,
		Timestamp: time.Now(),
	}
}
func Fail(code string, lang string, details ...ErrorDetail) HttpResult {
	codeDefine, ok := GetDefineCode(code, lang)
	msg := codeDefine.Message
	if !ok {
		msg = "The result code is not found."
	}

	return HttpResult{
		Message: msg,
		Status:  "error",
		Error: &HttpError{
			Code:             code,
			Message:          codeDefine.Message,
			DeveloperMessage: codeDefine.DeveloperMessage,
			Details:          details,
			Severity:         codeDefine.Severity,
			Action:           codeDefine.Action,
		},
		Timestamp: time.Now(),
	}
}
