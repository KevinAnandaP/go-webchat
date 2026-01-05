package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/vinneth/go-webchat/middleware"
	"github.com/vinneth/go-webchat/models"
	"github.com/vinneth/go-webchat/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AddContactRequest represents add contact payload
type AddContactRequest struct {
	UniqueID string `json:"unique_id"`
}

// AddContact adds a contact by unique ID
func AddContact(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req AddContactRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UniqueID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unique ID is required",
		})
	}

	// Find contact by unique ID
	contact, err := models.FindUserByUniqueID(req.UniqueID)
	if err != nil || contact == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User with this ID not found",
		})
	}

	// Cannot add self
	if contact.ID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You cannot add yourself as a contact",
		})
	}

	// Check if already a contact
	user, _ := models.FindUserByID(userID)
	if user != nil {
		for _, cID := range user.Contacts {
			if cID == contact.ID {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "Already in your contacts",
				})
			}
		}
	}

	// Add contact (both ways for mutual contact)
	if err := models.AddContact(userID, contact.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add contact",
		})
	}

	// Add reverse contact
	models.AddContact(contact.ID, userID)

	// Check if online
	isOnline := websocket.Hub.IsOnline(contact.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Contact added successfully",
		"contact": contact.ToPublic(isOnline),
	})
}

// GetContacts returns user's contact list
func GetContacts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	contacts, err := models.GetContacts(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch contacts",
		})
	}

	// Convert to public with online status
	publicContacts := make([]models.UserPublic, len(contacts))
	for i, contact := range contacts {
		isOnline := websocket.Hub.IsOnline(contact.ID)
		publicContacts[i] = contact.ToPublic(isOnline)
	}

	return c.JSON(fiber.Map{
		"contacts": publicContacts,
	})
}

// RemoveContact removes a contact
func RemoveContact(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	contactIDStr := c.Params("id")

	contactID, err := primitive.ObjectIDFromHex(contactIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid contact ID",
		})
	}

	// Remove contact both ways
	if err := models.RemoveContact(userID, contactID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove contact",
		})
	}

	models.RemoveContact(contactID, userID)

	return c.JSON(fiber.Map{
		"message": "Contact removed successfully",
	})
}

// SearchUserByUniqueID searches for a user by unique ID
func SearchUserByUniqueID(c *fiber.Ctx) error {
	uniqueID := c.Query("unique_id")
	if uniqueID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Unique ID query parameter is required",
		})
	}

	user, err := models.FindUserByUniqueID(uniqueID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	isOnline := websocket.Hub.IsOnline(user.ID)

	return c.JSON(fiber.Map{
		"user": user.ToPublic(isOnline),
	})
}
