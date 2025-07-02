package response

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type CodeDefinition struct {
	KO               *LangMessage `json:"ko"`
	EN               *LangMessage `json:"en"`
	DeveloperMessage string       `json:"developerMessage"`
	Service          string       `json:"service"`
	Type             string       `json:"type"`
	HttpStatus       int          `json:"httpStatus"`
	Severity         string       `json:"severity"`
	Action           string       `json:"action"`
	Message          string       `json:"message"`
}

type LangMessage struct {
	Message string `json:"message"`
	Action  string `json:"action"`
}

var globalErrorMessages map[string]CodeDefinition
var mu sync.RWMutex

func init() {
	if err := LoadErrorMessages(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load error messages on init: %v\n", err)
	}
}

func LoadErrorMessages() error { // TODO
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

	var loadedMessages map[string]CodeDefinition
	if err := json.Unmarshal(byteValue, &loadedMessages); err != nil {
		return fmt.Errorf("failed to unmarshal errors.json: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	globalErrorMessages = loadedMessages
	fmt.Println("Error messages reloaded successfully!")
	return nil
}

func GetDefineCode(code, lang string) (CodeDefinition, bool) {
	mu.RLock()
	defer mu.RUnlock()

	msg, ok := globalErrorMessages[code]
	if !ok {
		return CodeDefinition{
			KO:               &LangMessage{Message: "알 수 없는 오류가 발생했습니다.", Action: "잠시 후 다시 시도해주세요."},
			EN:               &LangMessage{Message: "An unknown error occurred.", Action: "Please try again later."},
			DeveloperMessage: "Error code not found in errors.json.",
			Service:          "System",
			Type:             "ErrorCodeNotFound",
			HttpStatus:       0,
			Severity:         "High",
		}, false
	}

	if lang == "ko" {
		msg.Message = msg.KO.Message
		msg.Action = msg.KO.Action
	} else if lang == "en" {
		msg.Message = msg.EN.Message
		msg.Action = msg.EN.Action
	}

	return msg, true
}
