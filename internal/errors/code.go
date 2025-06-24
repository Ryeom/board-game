package errors

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

const (
	ErrorCodeDefaultInternalServerError = "DEFAULT_INTERNAL_SERVER_ERROR"
)

// Authentication Error Codes
const (
	ErrorCodeAuthBind                      = "ERROR_AUTH_BIND"
	ErrorCodeAuthValidation                = "ERROR_AUTH_VALIDATION"
	ErrorCodeAuthEmailDuplicate            = "ERROR_AUTH_EMAIL_DUPLICATE"
	ErrorCodeAuthUserLookupFailed          = "ERROR_AUTH_USER_LOOKUP_FAILED"
	ErrorCodeAuthPasswordHashingFailed     = "ERROR_AUTH_PASSWORD_HASHING_FAILED"
	ErrorCodeAuthCreateUserFailed          = "ERROR_AUTH_CREATE_USER_FAILED"
	ErrorCodeAuthInvalidCredentials        = "ERROR_AUTH_INVALID_CREDENTIALS"
	ErrorCodeAuthJwtGenerationFailed       = "ERROR_AUTH_JWT_GENERATION_FAILED"
	ErrorCodeAuthInvalidToken              = "ERROR_AUTH_INVALID_TOKEN"
	ErrorCodeAuthLogoutFailed              = "ERROR_AUTH_LOGOUT_FAILED"
	ErrorCodeAuthTokenBlacklistCheckFailed = "ERROR_AUTH_TOKEN_BLACKLIST_CHECK_FAILED"
	ErrorCodeAuthTokenBlacklisted          = "ERROR_AUTH_TOKEN_BLACKLISTED"
)

// User Error Codes
const (
	ErrorCodeUserUnauthorized            = "ERROR_USER_UNAUTHORIZED"
	ErrorCodeUserProfileFetchFailed      = "ERROR_USER_PROFILE_FETCH_FAILED"
	ErrorCodeUserNotFound                = "ERROR_USER_NOT_FOUND"
	ErrorCodeUserProfileUpdateFailed     = "ERROR_USER_PROFILE_UPDATE_FAILED"
	ErrorCodeUserCurrentPasswordMismatch = "ERROR_USER_CURRENT_PASSWORD_MISMATCH"
	ErrorCodeUserPasswordChangeFailed    = "ERROR_USER_PASSWORD_CHANGE_FAILED"
)

// Room Error Codes
const (
	ErrorCodeRoomInvalidRequest      = "ERROR_ROOM_INVALID_REQUEST"
	ErrorCodeRoomNotFound            = "ERROR_ROOM_NOT_FOUND"
	ErrorCodeRoomUnsupportedGameMode = "ERROR_ROOM_UNSUPPORTED_GAME_MODE"
)

type ErrorMessage struct {
	KO               *LangMessage `json:"ko"`
	EN               *LangMessage `json:"en"`
	DeveloperMessage string       `json:"developerMessage"`
	Service          string       `json:"service"`
	Type             string       `json:"type"`
	HttpStatus       int          `json:"httpStatus"`
	Severity         string       `json:"severity"`
}

type LangMessage struct {
	Message string `json:"message"`
	Action  string `json:"action"`
}

var globalErrorMessages map[string]ErrorMessage
var mu sync.RWMutex

func init() {
	if err := LoadErrorMessages(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load error messages on init: %v\n", err)
	}
}

func LoadErrorMessages() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	jsonFilePath := filepath.Join(currentDir, "config", "errors.json")

	if _, err := os.Stat(jsonFilePath); os.IsNotExist(err) {
		jsonFilePath = filepath.Join(currentDir, "..", "..", "config", "errors.json")
		if _, err := os.Stat(jsonFilePath); os.IsNotExist(err) {
			return fmt.Errorf("error messages file not found at %s or %s", filepath.Join(currentDir, "config", "errors.json"), jsonFilePath)
		}
	}

	file, err := os.Open(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to open errors.json file at %s: %w", jsonFilePath, err)
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read errors.json file: %w", err)
	}

	var loadedMessages map[string]ErrorMessage
	if err := json.Unmarshal(byteValue, &loadedMessages); err != nil {
		return fmt.Errorf("failed to unmarshal errors.json: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	globalErrorMessages = loadedMessages
	fmt.Println("Error messages reloaded successfully!")
	return nil
}

func GetErrorMessage(code, lang string) (ErrorMessage, bool) {
	mu.RLock()
	defer mu.RUnlock()

	msg, ok := globalErrorMessages[code]
	if !ok {
		return ErrorMessage{
			KO:               &LangMessage{Message: "알 수 없는 오류가 발생했습니다.", Action: "잠시 후 다시 시도해주세요."},
			EN:               &LangMessage{Message: "An unknown error occurred.", Action: "Please try again later."},
			DeveloperMessage: "Error code not found in errors.json.",
			Service:          "System",
			Type:             "ErrorCodeNotFound",
			HttpStatus:       0,
			Severity:         "High",
		}, false
	}

	if lang == "ko" && msg.KO == nil {
		msg.KO = &LangMessage{Message: "알 수 없는 한국어 메시지.", Action: "관리자에게 문의하세요."}
	}
	if lang == "en" && msg.EN == nil {
		msg.EN = &LangMessage{Message: "Unknown English message.", Action: "Contact support."}
	}

	return msg, true
}
