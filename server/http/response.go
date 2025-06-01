package http

import "net/http"

func Success(data any) (int, map[string]any) {
	return http.StatusOK, map[string]any{
		"status":  "success",
		"data":    data,
		"message": nil,
	}
}

func Failure(message string, code int) (int, map[string]any) {
	return code, map[string]any{
		"status":  "error",
		"data":    nil,
		"message": message,
	}
}
