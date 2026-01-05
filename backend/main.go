package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/vinneth/go-webchat/config"
	"github.com/vinneth/go-webchat/database"
	"github.com/vinneth/go-webchat/handlers"
	"github.com/vinneth/go-webchat/middleware"
	ws "github.com/vinneth/go-webchat/websocket"
)

func main() {
	// Load configuration
	config.Load()

	// Connect to MongoDB
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer database.Disconnect()

	// Initialize WebSocket hub
	ws.InitHub()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Go WebChat",
		ErrorHandler: errorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} ${path}\n",
	}))

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     config.AppConfig.FrontendURL,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "go-webchat",
		})
	})

	// API routes
	api := app.Group("/api")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", handlers.Register)
	auth.Post("/login", handlers.Login)
	auth.Post("/logout", handlers.Logout)
	auth.Get("/google", handlers.GoogleLogin)
	auth.Get("/google/callback", handlers.GoogleCallback)

	// Protected auth routes
	auth.Get("/me", middleware.AuthRequired(), handlers.GetMe)
	auth.Put("/unique-id", middleware.AuthRequired(), handlers.UpdateUniqueID)

	// Contacts routes (protected)
	contacts := api.Group("/contacts", middleware.AuthRequired())
	contacts.Get("/", handlers.GetContacts)
	contacts.Post("/", handlers.AddContact)
	contacts.Delete("/:id", handlers.RemoveContact)
	contacts.Get("/search", handlers.SearchUserByUniqueID)

	// Conversations routes (protected)
	conversations := api.Group("/conversations", middleware.AuthRequired())
	conversations.Get("/", handlers.GetConversations)
	conversations.Post("/", handlers.CreateConversation)
	conversations.Get("/:id", handlers.GetConversation)
	conversations.Get("/:id/messages", handlers.GetMessages)

	// Groups routes (protected)
	groups := api.Group("/groups", middleware.AuthRequired())
	groups.Post("/", handlers.CreateGroup)
	groups.Put("/:id", handlers.UpdateGroup)
	groups.Post("/:id/members", handlers.AddGroupMember)
	groups.Delete("/:id/members/:userId", handlers.RemoveGroupMember)
	groups.Post("/:id/leave", handlers.LeaveGroup)

	// WebSocket route
	app.Use("/ws", ws.WebSocketUpgrade())
	app.Get("/ws", websocket.New(ws.HandleWebSocket))

	// Start server
	go func() {
		addr := ":" + config.AppConfig.Port
		log.Printf("🚀 Server starting on http://localhost%s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("Server stopped")
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   message,
		"success": false,
	})
}
