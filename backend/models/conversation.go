package models

import (
	"context"
	"time"

	"github.com/vinneth/go-webchat/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConversationType string

const (
	ConversationTypePrivate ConversationType = "private"
	ConversationTypeGroup   ConversationType = "group"
)

type Conversation struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Type      ConversationType     `bson:"type" json:"type"`
	Members   []primitive.ObjectID `bson:"members" json:"members"`
	GroupName string               `bson:"group_name,omitempty" json:"group_name,omitempty"`
	GroupIcon string               `bson:"group_icon,omitempty" json:"group_icon,omitempty"`
	Admin     primitive.ObjectID   `bson:"admin,omitempty" json:"admin,omitempty"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time            `bson:"updated_at" json:"updated_at"`
}

type ConversationWithDetails struct {
	Conversation
	LastMessage *Message     `json:"last_message,omitempty"`
	OtherUser   *UserPublic  `json:"other_user,omitempty"`   // For private chats
	MembersList []UserPublic `json:"members_list,omitempty"` // For group chats
	UnreadCount int          `json:"unread_count"`
}

// CreateConversation creates a new conversation
func CreateConversation(conv *Conversation) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()

	result, err := database.Conversations.InsertOne(ctx, conv)
	if err != nil {
		return err
	}

	conv.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindPrivateConversation finds an existing private conversation between two users
func FindPrivateConversation(user1ID, user2ID primitive.ObjectID) (*Conversation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var conv Conversation
	err := database.Conversations.FindOne(ctx, bson.M{
		"type": ConversationTypePrivate,
		"members": bson.M{
			"$all":  []primitive.ObjectID{user1ID, user2ID},
			"$size": 2,
		},
	}).Decode(&conv)

	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetOrCreatePrivateConversation gets or creates a private conversation
func GetOrCreatePrivateConversation(user1ID, user2ID primitive.ObjectID) (*Conversation, error) {
	conv, err := FindPrivateConversation(user1ID, user2ID)
	if err == nil && conv != nil {
		return conv, nil
	}

	// Create new conversation
	newConv := &Conversation{
		Type:    ConversationTypePrivate,
		Members: []primitive.ObjectID{user1ID, user2ID},
	}

	if err := CreateConversation(newConv); err != nil {
		return nil, err
	}

	return newConv, nil
}

// FindConversationByID finds a conversation by ID
func FindConversationByID(id primitive.ObjectID) (*Conversation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var conv Conversation
	err := database.Conversations.FindOne(ctx, bson.M{"_id": id}).Decode(&conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetUserConversations gets all conversations for a user
func GetUserConversations(userID primitive.ObjectID) ([]Conversation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"updated_at": -1})
	cursor, err := database.Conversations.Find(ctx, bson.M{
		"members": userID,
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversations []Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}

	return conversations, nil
}

// UpdateConversationTimestamp updates the conversation's updated_at field
func UpdateConversationTimestamp(convID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Conversations.UpdateOne(
		ctx,
		bson.M{"_id": convID},
		bson.M{"$set": bson.M{"updated_at": time.Now()}},
	)
	return err
}

// CreateGroup creates a new group conversation
func CreateGroup(name string, icon string, adminID primitive.ObjectID, memberIDs []primitive.ObjectID) (*Conversation, error) {
	// Ensure admin is in members
	members := append([]primitive.ObjectID{adminID}, memberIDs...)

	// Remove duplicates
	seen := make(map[primitive.ObjectID]bool)
	uniqueMembers := []primitive.ObjectID{}
	for _, id := range members {
		if !seen[id] {
			seen[id] = true
			uniqueMembers = append(uniqueMembers, id)
		}
	}

	conv := &Conversation{
		Type:      ConversationTypeGroup,
		Members:   uniqueMembers,
		GroupName: name,
		GroupIcon: icon,
		Admin:     adminID,
	}

	if err := CreateConversation(conv); err != nil {
		return nil, err
	}

	return conv, nil
}

// UpdateGroup updates group info
func UpdateGroup(convID primitive.ObjectID, name, icon string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"updated_at": time.Now()}
	if name != "" {
		update["group_name"] = name
	}
	if icon != "" {
		update["group_icon"] = icon
	}

	_, err := database.Conversations.UpdateOne(
		ctx,
		bson.M{"_id": convID},
		bson.M{"$set": update},
	)
	return err
}

// AddGroupMember adds a member to a group
func AddGroupMember(convID, memberID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Conversations.UpdateOne(
		ctx,
		bson.M{"_id": convID},
		bson.M{
			"$addToSet": bson.M{"members": memberID},
			"$set":      bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

// RemoveGroupMember removes a member from a group
func RemoveGroupMember(convID, memberID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Conversations.UpdateOne(
		ctx,
		bson.M{"_id": convID},
		bson.M{
			"$pull": bson.M{"members": memberID},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

// IsMember checks if a user is a member of a conversation
func IsMember(convID, userID primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := database.Conversations.CountDocuments(ctx, bson.M{
		"_id":     convID,
		"members": userID,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
