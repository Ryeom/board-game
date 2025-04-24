package log

import (
	"fmt"
	"github.com/labstack/echo/v4/middleware"
	"github.com/op/go-logging"
	"os"
	"strings"
)

var ServerLogDesc *os.File
var AccessLogDesc *os.File

const (
	ProjectName       = "hanabi_log"
	DefaultLogPath    = ""
	ServerLogFileName = "server.log"
	AccessLogFileName = "access.log"

	DashboardTimeFormat = "2006-01-02T15:04:05.999999"
)

var Logger *logging.Logger

func InitializeApplicationLog() error {
	var err error
	logPath := DefaultLogPath + ProjectName + "/"
	checkDirectoryPath(logPath)
	serverLogPath := logPath + ServerLogFileName
	checkFilePath(serverLogPath)
	accessLogPath := logPath + AccessLogFileName
	checkFilePath(accessLogPath)
	ServerLogDesc, err = os.OpenFile(serverLogPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	AccessLogDesc, err = os.OpenFile(accessLogPath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	Logger = logging.MustGetLogger(ProjectName)
	back1 := logging.NewLogBackend(ServerLogDesc, "", 0)
	format := logging.MustStringFormatter(`%{color}%{time:0102 15:04:05.000} %{shortfunc:15s} ▶ %{level:.5s}%{color:reset} %{shortfile:15s} %{message}`)
	back1Formatter := logging.NewBackendFormatter(back1, format)
	back1Leveled := logging.AddModuleLevel(back1) //기본로그 외에 추가로그를 남기는 로직
	back1Leveled.SetLevel(logging.ERROR, "")      //추가로그의 로그 기본 레벨

	logging.SetBackend(back1Formatter)
	logging.SetLevel(logging.DEBUG, ProjectName)

	Logger.Info(banner)
	Logger.Info("Process initialize ... Env :")
	return err
}

func checkDirectoryPath(dirPath string) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
}

func checkFilePath(filePath string) {
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		file, createErr := os.Create(filePath)
		if createErr != nil {
			panic(createErr)
		}
		fmt.Println("created ", file.Name())
	}
}

func CreateCustomLogConfig() middleware.LoggerConfig {
	return middleware.LoggerConfig{
		Skipper: middleware.DefaultSkipper,
		Format: `{"transaction_id":"${header:transaction-id}", "status_code":${status}, "E":"${error}"` +
			`, "REMOTE_ADDR":"${remote_ip}", "Client-Ip":"${header:Client-Ip}", "time":"${time_custom}", "return_time":"${latency_human}"` +
			`, "I":${bytes_in}, "O":${bytes_out}, "method":"${method}", "path":"${uri}"}` + "\n",
		CustomTimeFormat: DashboardTimeFormat,
		Output:           AccessLogDesc,
	}
}

var banner = `
` + strings.Repeat("░", 150) + `

` + strings.Repeat("▅", 150)
