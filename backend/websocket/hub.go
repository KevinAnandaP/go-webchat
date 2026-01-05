package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/vinneth/go-webchat/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID       primitive.ObjectID
	UserID   primitive.ObjectID
	Conn     WebSocketConn
	Hub      *WebSocketHub
	Send     chan []byte
	LastPing time.Time
}

// WebSocketConn interface for WebSocket connection
type WebSocketConn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

// WebSocketHub manages all WebSocket connections
type WebSocketHub struct {
	clients    map[primitive.ObjectID]map[*Client]bool // userID -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan BroadcastMessage
	mu         sync.RWMutex
}

// BroadcastMessage for sending to specific users
type BroadcastMessage struct {
	UserIDs []primitive.ObjectID
	Message []byte
}

// Hub is the global WebSocket hub
var Hub *WebSocketHub

// NewHub creates a new WebSocketHub
func NewHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[primitive.ObjectID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan BroadcastMessage, 256),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()

			// Notify contacts that user is online
			go h.notifyOnlineStatus(client.UserID, true)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.UserID)
						// Notify contacts that user is offline
						go h.notifyOnlineStatus(client.UserID, false)
						// Update last seen
						models.UpdateLastSeen(client.UserID)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			for _, userID := range message.UserIDs {
				h.mu.RLock()
				clients, ok := h.clients[userID]
				h.mu.RUnlock()
				if ok {
					for client := range clients {
						select {
						case client.Send <- message.Message:
						default:
							h.mu.Lock()
							close(client.Send)
							delete(h.clients[userID], client)
							h.mu.Unlock()
						}
					}
				}
			}
		}
	}
}

// Register adds a client to the hub
func (h *WebSocketHub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *WebSocketHub) Unregister(client *Client) {
	h.unregister <- client
}

// IsOnline checks if a user is online
func (h *WebSocketHub) IsOnline(userID primitive.ObjectID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients, ok := h.clients[userID]
	return ok && len(clients) > 0
}

// GetOnlineUsers returns list of online user IDs
func (h *WebSocketHub) GetOnlineUsers() []primitive.ObjectID {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := make([]primitive.ObjectID, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}

// SendToUser sends a message to all connections of a specific user
func (h *WebSocketHub) SendToUser(userID primitive.ObjectID, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.broadcast <- BroadcastMessage{
		UserIDs: []primitive.ObjectID{userID},
		Message: data,
	}
}

// SendToUsers sends a message to multiple users
func (h *WebSocketHub) SendToUsers(userIDs []primitive.ObjectID, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.broadcast <- BroadcastMessage{
		UserIDs: userIDs,
		Message: data,
	}
}

// BroadcastToConversation sends a message to all members of a conversation
func (h *WebSocketHub) BroadcastToConversation(convID primitive.ObjectID, msg WSMessage, excludeUserID *primitive.ObjectID) {
	conv, err := models.FindConversationByID(convID)
	if err != nil || conv == nil {
		return
	}

	userIDs := make([]primitive.ObjectID, 0, len(conv.Members))
	for _, memberID := range conv.Members {
		if excludeUserID == nil || memberID != *excludeUserID {
			userIDs = append(userIDs, memberID)
		}
	}

	h.SendToUsers(userIDs, msg)
}

// notifyOnlineStatus notifies contacts about user's online status
func (h *WebSocketHub) notifyOnlineStatus(userID primitive.ObjectID, isOnline bool) {
	contacts, err := models.GetContacts(userID)
	if err != nil {
		return
	}

	eventType := "user:offline"
	if isOnline {
		eventType = "user:online"
	}

	for _, contact := range contacts {
		h.SendToUser(contact.ID, WSMessage{
			Type: eventType,
			Payload: map[string]interface{}{
				"user_id": userID.Hex(),
			},
		})
	}
}

// InitHub initializes the global hub
func InitHub() {
	Hub = NewHub()
	go Hub.Run()
}
