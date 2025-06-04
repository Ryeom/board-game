package server

import (
	"errors"
	"fmt"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/Ryeom/board-game/log"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	execEnv    string
	envOnce    sync.Once
	configOnce sync.Once
)

func GetExecEnv() string {
	return execEnv
}

func SetEnv() error {
	var err error
	envOnce.Do(func() {
		if len(os.Args) != 2 {
			err = errors.New("invalid arg length: " + strconv.Itoa(len(os.Args)))
			return
		}
		allowed := map[string]bool{"local": true, "dev": true, "prod": true}
		arg := os.Args[1]
		if !allowed[arg] {
			err = errors.New("invalid env: " + arg)
			return
		}
		execEnv = arg
	})
	return err
}

func SetConfig() error {
	var err error
	configOnce.Do(func() {
		if err = loadConfigFile(""); err != nil {
			return
		}
		if err = decryptSensitiveValues(); err != nil {
			return
		}
		applyDefaultSettings()
	})
	return err
}

func loadConfigFile(path string) error {
	var fileName = "settings"
	var fileExt = "toml"
	var configPath string

	if path == "" {
		// 기본 경로: 현재 작업 디렉토리
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getwd failed: %w", err)
		}
		configPath = cwd
	} else {
		// 경로가 파일이면 이름, 확장자 분리
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		if info.IsDir() {
			configPath = path
		} else {
			configPath = filepath.Dir(path)
			fileName = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			fileExt = strings.TrimPrefix(filepath.Ext(path), ".")
		}
	}
	viper.SetConfigName(fileName)
	viper.SetConfigType(fileExt)
	viper.AddConfigPath(configPath)

	return viper.ReadInConfig()
}

func decryptSensitiveValues() error {
	mainKey := "bg."
	aesKey := []byte(viper.GetString(mainKey + "key"))
	isDesktop := isDesktopPlatform()

	for _, key := range viper.AllKeys() {
		if strings.HasPrefix(key, mainKey) {
			continue
		}
		val := viper.GetString(key)
		if val == "" {
			continue
		}
		decVal, err := util.DecryptAES(val, aesKey)
		if err != nil {
			return fmt.Errorf("decrypt '%s' failed: %w", key, err)
		}
		viper.Set(key, decVal)

		if isDesktop {
			log.Logger.Info("[set config]", key, ":", decVal)
		}
	}

	return nil
}
func isDesktopPlatform() bool {
	return runtime.GOOS == "windows" || runtime.GOOS == "darwin"
}

func applyDefaultSettings() {
	viper.SetDefault("bg.local-ip", util.GetLocalIP())
}
