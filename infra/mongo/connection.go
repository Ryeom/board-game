package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/Ryeom/board-game/log"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var Client *mongo.Client

const (
	DBName         = "board_game"
	ChatCollection = "chat_messages"
)

func Initialize() {
	ip := viper.GetString("mongo.ip")
	port := viper.GetString("mongo.port")
	user := viper.GetString("mongo.user")
	pw := viper.GetString("mongo.pw")

	Client = NewClient(ip, port, user, pw)
}
func NewClient(ip, port, user, pw string) *mongo.Client {
	var mongoURI string
	if user != "" && pw != "" {
		mongoURI = fmt.Sprintf("mongodb://%s:%s@%s:%s", user, pw, ip, port)
	} else {
		mongoURI = fmt.Sprintf("mongodb://%s:%s", ip, port)
	}

	clientOptions := options.Client().ApplyURI(mongoURI)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Logger.Fatalf("MongoDB 연결에 실패했습니다: %v", err)
		return nil
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Logger.Fatalf("MongoDB 서버에 Ping을 보낼 수 없습니다: %v", err)
		return nil
	}

	fmt.Println("MongoDB에 성공적으로 연결되었습니다!")
	return client
}

func GetCollection(collectionName string) *mongo.Collection {
	return Client.Database(DBName).Collection(collectionName)
}

func Disconnect(ctx context.Context) {
	if Client != nil {
		err := Client.Disconnect(ctx)
		if err != nil {
			log.Logger.Errorf("MongoDB Disconnect ERROR: %v", err)
		}
	}
}
