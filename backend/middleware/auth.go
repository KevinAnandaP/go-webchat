package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/vinneth/go-webchat/config"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID primitive.ObjectID, email string) (string, error) {
	claims := JWTClaims{
		UserID: userID.Hex(),
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.AppConfig.JWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// ValidateToken validates a JWT token and returns claims
func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// AuthRequired middleware checks for valid JWT token
func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var tokenString string

		// Try to get token from cookie first
		tokenString = c.Cookies("auth_token")

		// Fallback to Authorization header
		if tokenString == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Parse user ID
		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid user ID in token",
			})
		}

		// Set user info in context
		c.Locals("userID", userID)
		c.Locals("email", claims.Email)

		return c.Next()
	}
}

// GetUserID gets the authenticated user ID from context
func GetUserID(c *fiber.Ctx) primitive.ObjectID {
	userID, ok := c.Locals("userID").(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID
	}
	return userID
}

// SetAuthCookie sets the HTTP-only auth cookie
func SetAuthCookie(c *fiber.Ctx, token string, rememberMe bool) {
	maxAge := 24 * 60 * 60 // 24 hours
	if rememberMe {
		maxAge = 30 * 24 * 60 * 60 // 30 days
	}

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   config.AppConfig.Env == "production",
		HTTPOnly: true,
		SameSite: "Lax",
	})
}

// ClearAuthCookie clears the auth cookie
func ClearAuthCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HTTPOnly: true,
	})
}
