package db

import (
	"github.com/Ryeom/board-game/log"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// infra/db/db.go
var DB *gorm.DB

func Initialize() {
	dsn := viper.GetString("db.dsn")
	log.Logger.Debugf("DB connection initialized")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Logger.Fatalf("failed to connect: %v", err)
	}
	DB = db
}
