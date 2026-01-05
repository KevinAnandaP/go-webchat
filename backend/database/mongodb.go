package database

import (
	"context"
	"log"
	"time"

	"github.com/vinneth/go-webchat/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client   *mongo.Client
	Database *mongo.Database
)

// Collections
var (
	Users         *mongo.Collection
	Conversations *mongo.Collection
	Messages      *mongo.Collection
)

func Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.AppConfig.MongoDBURI)
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	Client = client
	Database = client.Database(config.AppConfig.MongoDBDatabase)

	// Initialize collections
	Users = Database.Collection("users")
	Conversations = Database.Collection("conversations")
	Messages = Database.Collection("messages")

	log.Println("✅ Connected to MongoDB Atlas")
	return nil
}

func Disconnect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if Client != nil {
		if err := Client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}
}
