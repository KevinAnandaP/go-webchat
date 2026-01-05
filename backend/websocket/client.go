package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/vinneth/go-webchat/middleware"
	"github.com/vinneth/go-webchat/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 4096
)

// FiberWebSocketConn wraps fiber websocket connection
type FiberWebSocketConn struct {
	*websocket.Conn
}

func (c *FiberWebSocketConn) WriteMessage(messageType int, data []byte) error {
	return c.Conn.WriteMessage(messageType, data)
}

func (c *FiberWebSocketConn) ReadMessage() (int, []byte, error) {
	return c.Conn.ReadMessage()
}

func (c *FiberWebSocketConn) Close() error {
	return c.Conn.Close()
}

// WebSocketUpgrade middleware to check WebSocket upgrade
func WebSocketUpgrade() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *websocket.Conn) {
	// Get user ID from query or locals
	tokenString := c.Query("token")
	if tokenString == "" {
		// Try from cookie
		tokenString = c.Cookies("auth_token")
	}

	if tokenString == "" {
		c.WriteJSON(WSMessage{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Authentication required",
			},
		})
		c.Close()
		return
	}

	claims, err := middleware.ValidateToken(tokenString)
	if err != nil {
		c.WriteJSON(WSMessage{
			Type: "error",
			Payload: map[string]interface{}{
				"message": "Invalid token",
			},
		})
		c.Close()
		return
	}

	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		c.Close()
		return
	}

	// Create client
	client := &Client{
		ID:       primitive.NewObjectID(),
		UserID:   userID,
		Conn:     &FiberWebSocketConn{c},
		Hub:      Hub,
		Send:     make(chan []byte, 256),
		LastPing: time.Now(),
	}

	// Register client
	Hub.Register(client)

	// Start write pump in goroutine
	go client.writePump()

	// Run read pump (blocking)
	client.readPump()
}

// readPump pumps messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(msg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(msg WSMessage) {
	switch msg.Type {
	case "ping":
		c.LastPing = time.Now()
		c.sendMessage(WSMessage{Type: "pong", Payload: map[string]interface{}{}})

	case "message:send":
		c.handleSendMessage(msg.Payload)

	case "typing:start":
		c.handleTyping(msg.Payload, true)

	case "typing:stop":
		c.handleTyping(msg.Payload, false)

	case "message:read":
		c.handleMessageRead(msg.Payload)
	}
}

// handleSendMessage handles sending a new message
func (c *Client) handleSendMessage(payload map[string]interface{}) {
	convIDStr, ok := payload["conversation_id"].(string)
	if !ok {
		return
	}
	content, ok := payload["content"].(string)
	if !ok || content == "" {
		return
	}

	convID, err := primitive.ObjectIDFromHex(convIDStr)
	if err != nil {
		return
	}

	// Verify user is member of conversation
	isMember, err := models.IsMember(convID, c.UserID)
	if err != nil || !isMember {
		return
	}

	// Create message
	msg := &models.Message{
		ConversationID: convID,
		SenderID:       c.UserID,
		Content:        content,
	}

	if err := models.CreateMessage(msg); err != nil {
		log.Printf("Failed to create message: %v", err)
		return
	}

	// Get sender info
	sender, _ := models.FindUserByID(c.UserID)
	var senderPublic *models.UserPublic
	if sender != nil {
		public := sender.ToPublic(true)
		senderPublic = &public
	}

	// Send confirmation to sender
	c.sendMessage(WSMessage{
		Type: "message:sent",
		Payload: map[string]interface{}{
			"temp_id":    payload["temp_id"], // For optimistic UI
			"message_id": msg.ID.Hex(),
			"status":     "sent",
		},
	})

	// Broadcast to conversation members
	Hub.BroadcastToConversation(convID, WSMessage{
		Type: "message:new",
		Payload: map[string]interface{}{
			"message": models.MessageWithSender{
				Message: *msg,
				Sender:  senderPublic,
			},
		},
	}, &c.UserID)
}

// handleTyping handles typing indicators
func (c *Client) handleTyping(payload map[string]interface{}, isTyping bool) {
	convIDStr, ok := payload["conversation_id"].(string)
	if !ok {
		return
	}

	convID, err := primitive.ObjectIDFromHex(convIDStr)
	if err != nil {
		return
	}

	eventType := "user:typing_stop"
	if isTyping {
		eventType = "user:typing"
	}

	Hub.BroadcastToConversation(convID, WSMessage{
		Type: eventType,
		Payload: map[string]interface{}{
			"conversation_id": convIDStr,
			"user_id":         c.UserID.Hex(),
		},
	}, &c.UserID)
}

// handleMessageRead handles read receipts
func (c *Client) handleMessageRead(payload map[string]interface{}) {
	convIDStr, ok := payload["conversation_id"].(string)
	if !ok {
		return
	}
	msgIDStr, _ := payload["message_id"].(string)

	convID, err := primitive.ObjectIDFromHex(convIDStr)
	if err != nil {
		return
	}

	if msgIDStr != "" {
		// Mark specific message as read
		msgID, err := primitive.ObjectIDFromHex(msgIDStr)
		if err != nil {
			return
		}
		models.MarkMessageAsRead(msgID, c.UserID)

		// Notify sender
		msg, _ := models.FindMessageByID(msgID)
		if msg != nil && msg.SenderID != c.UserID {
			Hub.SendToUser(msg.SenderID, WSMessage{
				Type: "message:status",
				Payload: map[string]interface{}{
					"message_id": msgIDStr,
					"status":     "read",
					"read_by":    c.UserID.Hex(),
				},
			})
		}
	} else {
		// Mark all messages in conversation as read
		models.MarkConversationAsRead(convID, c.UserID)
	}
}

// sendMessage sends a message to this client
func (c *Client) sendMessage(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case c.Send <- data:
	default:
		// Channel full, close connection
		c.Hub.Unregister(c)
	}
}
