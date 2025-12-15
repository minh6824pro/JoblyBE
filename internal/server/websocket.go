package server

import (
	"JobblyBE/pkg/middleware/auth"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin allows all origins for WebSocket connections
	// This works in conjunction with the gorilla/handlers CORS filter
	// In production, consider restricting to specific origins
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins - matches the CORS policy in http.go
		return true
	},
}

// WSMessageType represents the type of WebSocket message
type WSMessageType string

const (
	WSMessageTypeJobStatus    WSMessageType = "job_status"
	WSMessageTypeError        WSMessageType = "error"
	WSMessageTypePing         WSMessageType = "ping"
	WSMessageTypePong         WSMessageType = "pong"
	WSMessageTypeNotification WSMessageType = "notification"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    WSMessageType `json:"type"`
	Payload interface{}   `json:"payload"`
}

// JobStatusPayload represents the payload for job status updates
type JobStatusPayload struct {
	JobID        string      `json:"job_id"`
	Status       string      `json:"status"`
	ErrorMessage string      `json:"error_message,omitempty"`
	ResumeID     string      `json:"resume_id,omitempty"`
	CVData       interface{} `json:"cv_data,omitempty"`
}

// NotificationPayload represents the payload for notification messages
type NotificationPayload struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	ObjectID  string `json:"object_id,omitempty"`
	CreatedAt string `json:"created_at"`
}

// Client represents a WebSocket client
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients by userID
	clients map[string]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex

	log *log.Helper
}

// NewHub creates a new Hub instance
func NewHub(logger log.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		log:        log.NewHelper(logger),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.log.Infof("WebSocket client registered for user: %s, total clients: %d", client.userID, len(h.clients[client.userID]))
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.userID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.userID)
					}
					h.log.Infof("WebSocket client unregistered for user: %s", client.userID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// SendToUser sends a message to all clients of a specific user
func (h *Hub) SendToUser(userID string, message *WSMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		h.log.Errorf("Failed to marshal WebSocket message: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		h.log.Infof("No WebSocket clients found for user: %s", userID)
		return
	}

	for client := range clients {
		select {
		case client.send <- data:
			h.log.Infof("Message sent to user %s via WebSocket", userID)
		default:
			h.mu.Lock()
			delete(clients, client)
			close(client.send)
			h.mu.Unlock()
		}
	}
}

// SendJobStatus sends a job status update to a user
func (h *Hub) SendJobStatus(userID string, payload *JobStatusPayload) {
	// Log the payload being sent for debugging
	if payload.CVData != nil {
		h.log.Infof("SendJobStatus: Sending CVData to user %s for job %s", userID, payload.JobID)
	}

	message := &WSMessage{
		Type:    WSMessageTypeJobStatus,
		Payload: payload,
	}
	h.SendToUser(userID, message)
}

// SendNotification sends a notification to a user
func (h *Hub) SendNotification(userID string, payload *NotificationPayload) {
	h.log.Infof("SendNotification: Sending notification to user %s - %s", userID, payload.Title)

	message := &WSMessage{
		Type:    WSMessageTypeNotification,
		Payload: payload,
	}
	h.SendToUser(userID, message)
}

// GetConnectedUsers returns a list of all connected user IDs
func (h *Hub) GetConnectedUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}
	return users
}

// GetUserConnectionCount returns the number of connections for a specific user
func (h *Hub) GetUserConnectionCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		return len(clients)
	}
	return 0
}

// GetTotalConnections returns the total number of active connections
func (h *Hub) GetTotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.log.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// Handle ping messages from client
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg.Type == WSMessageTypePing {
				pongMsg := &WSMessage{Type: WSMessageTypePong}
				data, _ := json.Marshal(pongMsg)
				c.send <- data
			}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub       *Hub
	jwtSecret string
	log       *log.Helper
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub, jwtSecret string, logger log.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:       hub,
		jwtSecret: jwtSecret,
		log:       log.NewHelper(logger),
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from query param or header
	// In production, you should validate the JWT token
	token := r.URL.Query().Get("token")
	if token == "" {
		token = extractToken(r)
	}

	if token == "" {
		http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
		return
	}

	// Validate token and get user ID
	claims, err := auth.ValidateAccessToken(token, h.jwtSecret)
	if err != nil {
		h.log.Errorf("Invalid token: %v", err)
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	userID := claims.UserID
	if userID == "" {
		http.Error(w, "Unauthorized: user ID not found", http.StatusUnauthorized)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:    h.hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}

	h.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}
