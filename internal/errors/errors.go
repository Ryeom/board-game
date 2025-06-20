package errors

import (
	"errors"
	"fmt"
)

type AppError struct {
	Status   int    `json:"status"`         // HTTP 상태 코드
	Code     string `json:"code,omitempty"` // 내부 오류 코드 (JSON 키와 동일)
	Message  string `json:"message"`        // 사용자에게 보여줄 메시지
	Severity string `json:"severity,omitempty"`
	Type     string `json:"type,omitempty"`

	Err error `json:"-"` // 실제 원본 오류
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("status: %d, code: %s, message: %s, original error: %v", e.Status, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("status: %d, code: %s, message: %s", e.Status, e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(status int, code, message string, err error) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func NewAppErrorFromCode(errorCode, lang string, originalErr error) *AppError {
	errMsg, found := GetErrorMessage(errorCode, lang)

	if !found {
		defaultMsg, _ := GetErrorMessage("DEFAULT_INTERNAL_SERVER_ERROR", lang)
		if defaultMsg.KO == nil {
			return InternalServerError("알 수 없는 서버 오류가 발생했습니다.", originalErr)
		}
		return InternalServerError(defaultMsg.KO.Message, originalErr)
	}

	var displayMessage string
	if lang == "ko" && errMsg.KO != nil {
		displayMessage = errMsg.KO.Message
	} else {
		displayMessage = errMsg.EN.Message
	}

	return &AppError{
		Status:  errMsg.HttpStatus,
		Code:    errorCode,
		Message: displayMessage,
		Err:     originalErr,
	}
}

func BadRequest(code string, err error) *AppError {
	// "ERROR_AUTH_BIND", "ERROR_ROOM_INVALID_REQUEST" 등
	return NewAppErrorFromCode(code, "ko", err)
}

func Unauthorized(code string, err error) *AppError {
	// "ERROR_AUTH_INVALID_CREDENTIALS", "ERROR_USER_UNAUTHORIZED" 등
	return NewAppErrorFromCode(code, "ko", err)
}

func Forbidden(code string, err error) *AppError {
	// 이 코드는 현재 JSON에 없지만, 추가될 경우를 대비
	return NewAppErrorFromCode(code, "ko", err)
}

func NotFound(code string, err error) *AppError {
	// "ERROR_USER_NOT_FOUND", "ERROR_ROOM_NOT_FOUND" 등
	return NewAppErrorFromCode(code, "ko", err)
}

func Conflict(code string, err error) *AppError {
	// "ERROR_AUTH_EMAIL_DUPLICATE" 등
	return NewAppErrorFromCode(code, "ko", err)
}

func InternalServerError(code string, err error) *AppError {
	// "ERROR_AUTH_USER_LOOKUP_FAILED", "ERROR_AUTH_PASSWORD_HASHING_FAILED" 등
	return NewAppErrorFromCode(code, "ko", err)
}

func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}
