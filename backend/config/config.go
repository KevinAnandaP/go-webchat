package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	Env             string
	MongoDBURI      string
	MongoDBDatabase string
	JWTSecret       string
	JWTExpiry       time.Duration
	GoogleClientID  string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL     string
}

var AppConfig *Config

func Load() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))
	if err != nil {
		jwtExpiry = 24 * time.Hour
	}

	AppConfig = &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             getEnv("ENV", "development"),
		MongoDBURI:      getEnv("MONGODB_URI", ""),
		MongoDBDatabase: getEnv("MONGODB_DATABASE", "go_webchat"),
		JWTSecret:       getEnv("JWT_SECRET", "default-secret-key"),
		JWTExpiry:       jwtExpiry,
		GoogleClientID:  getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/auth/google/callback"),
		FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:3000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
