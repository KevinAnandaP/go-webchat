package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/vinneth/go-webchat/middleware"
	"github.com/vinneth/go-webchat/models"
	"github.com/vinneth/go-webchat/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateConversationRequest represents create conversation payload
type CreateConversationRequest struct {
	UserID string `json:"user_id"` // For private chat
}

// GetConversations returns user's conversations
func GetConversations(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	conversations, err := models.GetUserConversations(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch conversations",
		})
	}

	// Enrich with details
	result := make([]models.ConversationWithDetails, 0, len(conversations))
	for _, conv := range conversations {
		details := models.ConversationWithDetails{
			Conversation: conv,
		}

		// Get last message
		lastMsg, _ := models.GetLastMessage(conv.ID)
		details.LastMessage = lastMsg

		// Get unread count
		unreadCount, _ := models.GetUnreadCount(conv.ID, userID)
		details.UnreadCount = int(unreadCount)

		if conv.Type == models.ConversationTypePrivate {
			// Get other user for private chat
			for _, memberID := range conv.Members {
				if memberID != userID {
					otherUser, _ := models.FindUserByID(memberID)
					if otherUser != nil {
						isOnline := websocket.Hub.IsOnline(otherUser.ID)
						public := otherUser.ToPublic(isOnline)
						details.OtherUser = &public
					}
					break
				}
			}
		} else {
			// Get members list for group
			membersList := make([]models.UserPublic, 0)
			for _, memberID := range conv.Members {
				member, _ := models.FindUserByID(memberID)
				if member != nil {
					isOnline := websocket.Hub.IsOnline(member.ID)
					membersList = append(membersList, member.ToPublic(isOnline))
				}
			}
			details.MembersList = membersList
		}

		result = append(result, details)
	}

	return c.JSON(fiber.Map{
		"conversations": result,
	})
}

// CreateConversation creates a new private conversation
func CreateConversation(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req CreateConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	otherUserID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	// Cannot create conversation with self
	if otherUserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot create conversation with yourself",
		})
	}

	// Check if other user exists
	otherUser, err := models.FindUserByID(otherUserID)
	if err != nil || otherUser == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Get or create conversation
	conv, err := models.GetOrCreatePrivateConversation(userID, otherUserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create conversation",
		})
	}

	// Return with details
	isOnline := websocket.Hub.IsOnline(otherUser.ID)
	result := models.ConversationWithDetails{
		Conversation: *conv,
		OtherUser:    &models.UserPublic{ID: otherUser.ID, UniqueID: otherUser.UniqueID, Name: otherUser.Name, Avatar: otherUser.Avatar, IsOnline: isOnline},
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"conversation": result,
	})
}

// GetConversation returns a specific conversation
func GetConversation(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	convIDStr := c.Params("id")

	convID, err := primitive.ObjectIDFromHex(convIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	// Check if user is member
	isMember, err := models.IsMember(convID, userID)
	if err != nil || !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	conv, err := models.FindConversationByID(convID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Conversation not found",
		})
	}

	// Build details
	details := models.ConversationWithDetails{
		Conversation: *conv,
	}

	if conv.Type == models.ConversationTypePrivate {
		for _, memberID := range conv.Members {
			if memberID != userID {
				otherUser, _ := models.FindUserByID(memberID)
				if otherUser != nil {
					isOnline := websocket.Hub.IsOnline(otherUser.ID)
					public := otherUser.ToPublic(isOnline)
					details.OtherUser = &public
				}
				break
			}
		}
	} else {
		membersList := make([]models.UserPublic, 0)
		for _, memberID := range conv.Members {
			member, _ := models.FindUserByID(memberID)
			if member != nil {
				isOnline := websocket.Hub.IsOnline(member.ID)
				membersList = append(membersList, member.ToPublic(isOnline))
			}
		}
		details.MembersList = membersList
	}

	return c.JSON(fiber.Map{
		"conversation": details,
	})
}

// GetMessages returns messages for a conversation
func GetMessages(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	convIDStr := c.Params("id")

	convID, err := primitive.ObjectIDFromHex(convIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	// Check if user is member
	isMember, err := models.IsMember(convID, userID)
	if err != nil || !isMember {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Pagination
	limit, _ := strconv.ParseInt(c.Query("limit", "50"), 10, 64)
	skip, _ := strconv.ParseInt(c.Query("skip", "0"), 10, 64)

	if limit > 100 {
		limit = 100
	}

	messages, err := models.GetMessages(convID, limit, skip)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch messages",
		})
	}

	// Mark as read
	models.MarkConversationAsRead(convID, userID)

	// Enrich with sender info
	result := make([]models.MessageWithSender, len(messages))
	for i, msg := range messages {
		result[i] = models.MessageWithSender{
			Message: msg,
		}
		sender, _ := models.FindUserByID(msg.SenderID)
		if sender != nil {
			isOnline := websocket.Hub.IsOnline(sender.ID)
			public := sender.ToPublic(isOnline)
			result[i].Sender = &public
		}
	}

	return c.JSON(fiber.Map{
		"messages": result,
	})
}
