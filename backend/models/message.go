package models

import (
	"context"
	"time"

	"github.com/vinneth/go-webchat/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

type Message struct {
	ID             primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	ConversationID primitive.ObjectID   `bson:"conversation_id" json:"conversation_id"`
	SenderID       primitive.ObjectID   `bson:"sender_id" json:"sender_id"`
	Content        string               `bson:"content" json:"content"`
	Status         MessageStatus        `bson:"status" json:"status"`
	ReadBy         []primitive.ObjectID `bson:"read_by" json:"read_by"`
	CreatedAt      time.Time            `bson:"created_at" json:"created_at"`
}

type MessageWithSender struct {
	Message
	Sender *UserPublic `json:"sender,omitempty"`
}

// CreateMessage creates a new message
func CreateMessage(msg *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg.CreatedAt = time.Now()
	msg.Status = MessageStatusSent
	msg.ReadBy = []primitive.ObjectID{msg.SenderID}

	result, err := database.Messages.InsertOne(ctx, msg)
	if err != nil {
		return err
	}

	msg.ID = result.InsertedID.(primitive.ObjectID)

	// Update conversation timestamp
	UpdateConversationTimestamp(msg.ConversationID)

	return nil
}

// GetMessages gets messages for a conversation with pagination
func GetMessages(conversationID primitive.ObjectID, limit, skip int64) ([]Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().
		SetSort(bson.M{"created_at": -1}).
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := database.Messages.Find(ctx, bson.M{
		"conversation_id": conversationID,
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetLastMessage gets the last message for a conversation
func GetLastMessage(conversationID primitive.ObjectID) (*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.FindOne().SetSort(bson.M{"created_at": -1})

	var msg Message
	err := database.Messages.FindOne(ctx, bson.M{
		"conversation_id": conversationID,
	}, opts).Decode(&msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// MarkMessageAsRead marks a message as read by a user
func MarkMessageAsRead(msgID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Messages.UpdateOne(
		ctx,
		bson.M{"_id": msgID},
		bson.M{
			"$addToSet": bson.M{"read_by": userID},
			"$set":      bson.M{"status": MessageStatusRead},
		},
	)
	return err
}

// MarkConversationAsRead marks all messages in a conversation as read by a user
func MarkConversationAsRead(conversationID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Messages.UpdateMany(
		ctx,
		bson.M{
			"conversation_id": conversationID,
			"sender_id":       bson.M{"$ne": userID},
		},
		bson.M{
			"$addToSet": bson.M{"read_by": userID},
		},
	)
	return err
}

// UpdateMessageStatus updates message delivery status
func UpdateMessageStatus(msgID primitive.ObjectID, status MessageStatus) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Messages.UpdateOne(
		ctx,
		bson.M{"_id": msgID},
		bson.M{"$set": bson.M{"status": status}},
	)
	return err
}

// GetUnreadCount gets unread message count for a user in a conversation
func GetUnreadCount(conversationID, userID primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := database.Messages.CountDocuments(ctx, bson.M{
		"conversation_id": conversationID,
		"sender_id":       bson.M{"$ne": userID},
		"read_by":         bson.M{"$nin": []primitive.ObjectID{userID}},
	})
	return count, err
}

// FindMessageByID finds a message by ID
func FindMessageByID(id primitive.ObjectID) (*Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var msg Message
	err := database.Messages.FindOne(ctx, bson.M{"_id": id}).Decode(&msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
