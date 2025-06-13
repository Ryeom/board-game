package test

import (
	"fmt"
	"github.com/Ryeom/board-game/internal/util"
	"github.com/spf13/viper"
	"strings"
	"testing"
)

var key = "HelloooBoardGame"

func TestEncryption(t *testing.T) {

	list := map[string]string{}
	for k, v := range list {
		encValue, _ := util.EncryptAES(v, []byte(key))
		//_, _ := EncryptAES(k, []byte(Key))
		fmt.Println(k, " = ", encValue)
		//fmt.Println(k, " = ", `"`+encValue+`"`)
	}

}
func TestDecryption(t *testing.T) {
	list := map[string]string{}
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
		decValue, err := util.DecryptAES(v, []byte(key))
		if err != nil {
			t.Errorf(v, err.Error())
		}
		fmt.Println(k, " = ", decValue)
	}

}
