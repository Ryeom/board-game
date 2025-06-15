package errors

import (
	"errors"
	"fmt"
	"net/http" // HTTP 상태 코드 사용
)

type AppError struct {
	Status  int    `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// Error 인터페이스를 구현합니다.
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

func BadRequest(message string, err error) *AppError {
	return NewAppError(http.StatusBadRequest, "BAD_REQUEST", message, err)
}

func Unauthorized(message string, err error) *AppError {
	return NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", message, err)
}

func Forbidden(message string, err error) *AppError {
	return NewAppError(http.StatusForbidden, "FORBIDDEN", message, err)
}

func NotFound(message string, err error) *AppError {
	return NewAppError(http.StatusNotFound, "NOT_FOUND", message, err)
}

func Conflict(message string, err error) *AppError {
	return NewAppError(http.StatusConflict, "CONFLICT", message, err)
}

func InternalServerError(message string, err error) *AppError {
	return NewAppError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message, err)
}

func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}
