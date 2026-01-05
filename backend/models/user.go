package models

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/vinneth/go-webchat/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID              primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UniqueID        string               `bson:"unique_id" json:"unique_id"`
	UniqueIDChanged bool                 `bson:"unique_id_changed" json:"unique_id_changed"`
	Email           string               `bson:"email" json:"email"`
	PasswordHash    string               `bson:"password_hash,omitempty" json:"-"`
	Name            string               `bson:"name" json:"name"`
	Avatar          string               `bson:"avatar" json:"avatar"`
	AuthProvider    string               `bson:"auth_provider" json:"auth_provider"` // "local" or "google"
	Contacts        []primitive.ObjectID `bson:"contacts" json:"contacts"`
	CreatedAt       time.Time            `bson:"created_at" json:"created_at"`
	LastSeen        time.Time            `bson:"last_seen" json:"last_seen"`
}

type UserPublic struct {
	ID       primitive.ObjectID `json:"id"`
	UniqueID string             `json:"unique_id"`
	Name     string             `json:"name"`
	Avatar   string             `json:"avatar"`
	LastSeen time.Time          `json:"last_seen"`
	IsOnline bool               `json:"is_online"`
}

// GenerateUniqueID creates a unique ID like #GOPRO-882
func GenerateUniqueID() (string, error) {
	prefixes := []string{"CHAT", "USER", "TALK", "GOPRO", "WAVE", "PING"}
	
	// Random prefix
	prefixIdx, err := rand.Int(rand.Reader, big.NewInt(int64(len(prefixes))))
	if err != nil {
		return "", err
	}
	prefix := prefixes[prefixIdx.Int64()]
	
	// Random 3-digit number
	num, err := rand.Int(rand.Reader, big.NewInt(900))
	if err != nil {
		return "", err
	}
	number := num.Int64() + 100 // 100-999
	
	uniqueID := fmt.Sprintf("#%s-%d", prefix, number)
	
	// Check if exists, regenerate if needed
	ctx := context.Background()
	count, _ := database.Users.CountDocuments(ctx, bson.M{"unique_id": uniqueID})
	if count > 0 {
		return GenerateUniqueID() // Recurse to generate new one
	}
	
	return uniqueID, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPassword compares password with hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateUser creates a new user in the database
func CreateUser(user *User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user.CreatedAt = time.Now()
	user.LastSeen = time.Now()
	user.Contacts = []primitive.ObjectID{}

	if user.UniqueID == "" {
		uniqueID, err := GenerateUniqueID()
		if err != nil {
			return err
		}
		user.UniqueID = uniqueID
	}

	result, err := database.Users.InsertOne(ctx, user)
	if err != nil {
		return err
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindUserByEmail finds a user by email
func FindUserByEmail(email string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := database.Users.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByID finds a user by ObjectID
func FindUserByID(id primitive.ObjectID) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := database.Users.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// FindUserByUniqueID finds a user by unique ID
func FindUserByUniqueID(uniqueID string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := database.Users.FindOne(ctx, bson.M{"unique_id": uniqueID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateLastSeen updates the user's last seen timestamp
func UpdateLastSeen(userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Users.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"last_seen": time.Now()}},
	)
	return err
}

// AddContact adds a contact to user's contact list
func AddContact(userID, contactID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Users.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$addToSet": bson.M{"contacts": contactID}},
	)
	return err
}

// RemoveContact removes a contact from user's contact list
func RemoveContact(userID, contactID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := database.Users.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$pull": bson.M{"contacts": contactID}},
	)
	return err
}

// GetContacts gets all contacts for a user
func GetContacts(userID primitive.ObjectID) ([]User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	user, err := FindUserByID(userID)
	if err != nil || user == nil {
		return nil, err
	}

	if len(user.Contacts) == 0 {
		return []User{}, nil
	}

	cursor, err := database.Users.Find(ctx, bson.M{"_id": bson.M{"$in": user.Contacts}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var contacts []User
	if err := cursor.All(ctx, &contacts); err != nil {
		return nil, err
	}

	return contacts, nil
}

// ToPublic converts User to UserPublic (safe for client)
func (u *User) ToPublic(isOnline bool) UserPublic {
	return UserPublic{
		ID:       u.ID,
		UniqueID: u.UniqueID,
		Name:     u.Name,
		Avatar:   u.Avatar,
		LastSeen: u.LastSeen,
		IsOnline: isOnline,
	}
}
