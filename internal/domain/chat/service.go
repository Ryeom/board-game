package chat

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Ryeom/board-game/infra/mongo"
	"github.com/Ryeom/board-game/log"
)

const MaxChatHistory = 50

func SaveChatMessage(ctx context.Context, roomID string, record *ChatRecord) error {
	collection := mongo.GetCollection(mongo.ChatCollection)

	messageDoc := bson.M{
		"roomId":     roomID,
		"senderId":   record.SenderID,
		"senderName": record.SenderName,
		"message":    record.Message,
		"timestamp":  record.Timestamp,
	}

	_, err := collection.InsertOne(ctx, messageDoc)
	if err != nil {
		log.Logger.Errorf("chat.Service - Failed to insert chat message into MongoDB for room %s: %v", roomID, err)
		return fmt.Errorf("failed to save chat message to MongoDB: %w", err)
	}

	return nil
}

func GetChatHistory(ctx context.Context, roomID string) ([]*ChatRecord, error) {
	collection := mongo.GetCollection(mongo.ChatCollection)

	filter := bson.M{"roomId": roomID}

	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "timestamp", Value: -1}})
	findOptions.SetLimit(MaxChatHistory)

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Logger.Errorf("chat.Service - Failed to retrieve chat history from MongoDB for room %s: %v", roomID, err)
		return nil, fmt.Errorf("failed to retrieve chat history from MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	var chatRecords []*ChatRecord
	for cursor.Next(ctx) {
		var record ChatRecord
		if err = cursor.Decode(&record); err != nil {
			log.Logger.Errorf("chat.Service - Failed to decode chat record from MongoDB: %v", err)
			continue
		}
		chatRecords = append(chatRecords, &record)
	}

	if err = cursor.Err(); err != nil {
		log.Logger.Errorf("chat.Service - Cursor error during chat history retrieval: %v", err)
		return nil, fmt.Errorf("cursor error during chat history retrieval: %w", err)
	}

	for i, j := 0, len(chatRecords)-1; i < j; i, j = i+1, j-1 {
		chatRecords[i], chatRecords[j] = chatRecords[j], chatRecords[i]
	}

	return chatRecords, nil
}
