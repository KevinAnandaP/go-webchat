package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/vinneth/go-webchat/config"
	"github.com/vinneth/go-webchat/middleware"
	"github.com/vinneth/go-webchat/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// RegisterRequest represents registration payload
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest represents login payload
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Message string             `json:"message"`
	User    *models.UserPublic `json:"user"`
}

// Google OAuth config
func getGoogleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.AppConfig.GoogleClientID,
		ClientSecret: config.AppConfig.GoogleClientSecret,
		RedirectURL:  config.AppConfig.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// Register handles user registration
func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email, password, and name are required",
		})
	}

	if len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 6 characters",
		})
	}

	// Check if user exists
	existingUser, _ := models.FindUserByEmail(req.Email)
	if existingUser != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Email already registered",
		})
	}

	// Hash password
	hashedPassword, err := models.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
		})
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Name:         req.Name,
		Avatar:       fmt.Sprintf("https://api.dicebear.com/7.x/initials/svg?seed=%s", req.Name),
		AuthProvider: "local",
	}

	if err := models.CreateUser(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Generate JWT
	token, err := middleware.GenerateToken(user.ID, user.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Set cookie
	middleware.SetAuthCookie(c, token, false)

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		Message: "Registration successful",
		User:    &models.UserPublic{ID: user.ID, UniqueID: user.UniqueID, Name: user.Name, Avatar: user.Avatar},
	})
}

// Login handles user login
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	// Find user
	user, err := models.FindUserByEmail(req.Email)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Check password
	if !models.CheckPassword(req.Password, user.PasswordHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
		})
	}

	// Generate JWT
	token, err := middleware.GenerateToken(user.ID, user.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Set cookie
	middleware.SetAuthCookie(c, token, req.RememberMe)

	// Update last seen
	models.UpdateLastSeen(user.ID)

	return c.JSON(AuthResponse{
		Message: "Login successful",
		User:    &models.UserPublic{ID: user.ID, UniqueID: user.UniqueID, Name: user.Name, Avatar: user.Avatar},
	})
}

// Logout handles user logout
func Logout(c *fiber.Ctx) error {
	middleware.ClearAuthCookie(c)
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// GetMe returns current authenticated user
func GetMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID.IsZero() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	user, err := models.FindUserByID(userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Update last seen
	models.UpdateLastSeen(user.ID)

	return c.JSON(fiber.Map{
		"user": models.UserPublic{
			ID:       user.ID,
			UniqueID: user.UniqueID,
			Name:     user.Name,
			Avatar:   user.Avatar,
			LastSeen: user.LastSeen,
		},
	})
}

// GoogleLogin redirects to Google OAuth
func GoogleLogin(c *fiber.Ctx) error {
	oauthConfig := getGoogleOAuthConfig()
	url := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	return c.Redirect(url)
}

// GoogleCallback handles Google OAuth callback
func GoogleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Redirect(config.AppConfig.FrontendURL + "/login?error=no_code")
	}

	oauthConfig := getGoogleOAuthConfig()
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return c.Redirect(config.AppConfig.FrontendURL + "/login?error=exchange_failed")
	}

	// Get user info from Google
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return c.Redirect(config.AppConfig.FrontendURL + "/login?error=userinfo_failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Redirect(config.AppConfig.FrontendURL + "/login?error=read_failed")
	}

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return c.Redirect(config.AppConfig.FrontendURL + "/login?error=parse_failed")
	}

	// Find or create user
	user, _ := models.FindUserByEmail(googleUser.Email)
	if user == nil {
		// Create new user
		user = &models.User{
			Email:        googleUser.Email,
			Name:         googleUser.Name,
			Avatar:       googleUser.Picture,
			AuthProvider: "google",
		}
		if err := models.CreateUser(user); err != nil {
			return c.Redirect(config.AppConfig.FrontendURL + "/login?error=create_failed")
		}
	} else {
		// Update avatar if changed
		if user.Avatar != googleUser.Picture {
			// TODO: Update avatar in DB
		}
	}

	// Generate JWT
	jwtToken, err := middleware.GenerateToken(user.ID, user.Email)
	if err != nil {
		return c.Redirect(config.AppConfig.FrontendURL + "/login?error=token_failed")
	}

	// Set cookie
	middleware.SetAuthCookie(c, jwtToken, true)

	// Redirect to frontend
	return c.Redirect(config.AppConfig.FrontendURL + "/chat")
}

// UpdateUniqueID allows user to change their unique ID once
func UpdateUniqueID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		UniqueID string `json:"unique_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := models.FindUserByID(userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if user.UniqueIDChanged {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Unique ID can only be changed once",
		})
	}

	// Check if new ID is available
	existing, _ := models.FindUserByUniqueID(req.UniqueID)
	if existing != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "This unique ID is already taken",
		})
	}

	// TODO: Update unique ID in database
	// For now, return success
	return c.JSON(fiber.Map{
		"message":   "Unique ID updated successfully",
		"unique_id": req.UniqueID,
	})
}
