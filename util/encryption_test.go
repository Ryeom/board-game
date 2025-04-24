package util

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"testing"
)

var filePath = ""
var keyName = "bg.key"

func TestEncryption(t *testing.T) {

	list := map[string]string{
		"": "",
	}
	err := loadConfigFile(filePath)
	if err != nil {
		t.Errorf("Error loading config file: %v", err)
	}
	for k, v := range list {
		encValue, _ := EncryptAES(v, []byte(viper.GetString(keyName)))
		//_, _ := EncryptAES(k, []byte(Key))
		fmt.Println(k, " = ", encValue)
		//fmt.Println(k, " = ", `"`+encValue+`"`)
	}

}
func TestDecryption(t *testing.T) {
	list := map[string]string{
		"": "",
	}
	err := loadConfigFile(filePath)
	if err != nil {
		t.Errorf("Error loading config file: %v", err)
	}

	for _, key := range viper.AllKeys() {
		val := viper.GetString(key)
		if val == "" {
			continue
		}
		list[key] = val
	}

	for k, v := range list {
		var err error
		if v == "" || strings.HasPrefix(k, "bg.") {
			continue
		}
		decValue, err := DecryptAES(v, []byte(viper.GetString(keyName)))
		if err != nil {
			t.Errorf(v, err.Error())
		}
		fmt.Println(k, " = ", decValue)
	}

}
