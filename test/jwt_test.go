package test

import (
	"encoding/base64"
	"fmt"
	"github.com/Ryeom/board-game/internal/util"
	"math/rand"
	"os"
	"testing"
)

func TestJwtCreate(t *testing.T) {
	aesKey := ""

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	randomString := base64.URLEncoding.EncodeToString(b)
	fmt.Println(randomString)
	rawJwtSecret := randomString // 암호화 되지않은 JWT 키

	encryptedJwtSecret, err := util.EncryptAES(rawJwtSecret, []byte(aesKey)) //
	if err != nil {
		fmt.Printf("Error encrypting JWT secret: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("--- Generated Encrypted JWT Secret ---")
	fmt.Printf("Raw JWT Secret: %s\n", rawJwtSecret)
	fmt.Printf("Encrypted JWT Secret for settings.toml: \"%s\"\n", encryptedJwtSecret)
	fmt.Println("--------------------------------------")
	fmt.Println("\nCopy the 'Encrypted JWT Secret' (including quotes) and paste it into your settings.toml under [jwt] secret = ...")

	decryptedJwtSecret, err := util.DecryptAES(encryptedJwtSecret, []byte(aesKey))
	if err != nil {
		fmt.Printf("Error decrypting for verification: %s\n", err)
	} else {
		fmt.Printf("Decrypted (for verification): %s (Matches raw: %t)\n", decryptedJwtSecret, decryptedJwtSecret == rawJwtSecret)
	}
}
