package http

import "net/http" // http 패키지 임포트 필요

// HttpResult는 모든 HTTP 응답의 표준 형식을 정의합니다.
// 이 구조체는 Swagger 문서화를 위해서도 사용됩니다.
type HttpResult struct {
	Code   int         `json:"code"`           // HTTP 상태 코드 (예: 200, 400, 500)
	Msg    string      `json:"message"`        // 클라이언트에게 보여줄 메시지
	Data   interface{} `json:"data,omitempty"` // 실제 응답 데이터 (선택 사항)
	Status string      `json:"status"`         // 응답 상태 (예: "success", "error")
}

// Success는 성공적인 HTTP 응답을 생성합니다.
// 이제 HttpResult 구조체를 직접 반환하여 JSON 직렬화에 용이하게 합니다.
func Success(data any, message string) HttpResult { // message 인자 추가 (선택 사항)
	return HttpResult{
		Code:   http.StatusOK,
		Msg:    message, // 메시지 추가
		Data:   data,
		Status: "success",
	}
}

// Failure는 실패한 HTTP 응답을 생성합니다.
func Failure(message string, statusCode int) HttpResult {
	return HttpResult{
		Code:   statusCode,
		Msg:    message,
		Data:   nil,
		Status: "error",
	}
}
