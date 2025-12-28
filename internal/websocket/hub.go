package websocket

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypeNewMessage  MessageType = "new_message"
	MessageTypeError       MessageType = "error"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      MessageType `json:"type"`
	MailboxID uint        `json:"mailbox_id,omitempty"`
	Message   interface{} `json:"message,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// NewMessagePayload represents the payload for new message notifications
type NewMessagePayload struct {
	ID          uint   `json:"id"`
	SenderEmail string `json:"sender_email"`
	SenderName  string `json:"sender_name,omitempty"`
	Subject     string `json:"subject,omitempty"`
	ReceivedAt  string `json:"received_at"`
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Mailbox subscriptions: mailboxID -> set of clients
	subscriptions map[uint]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Subscribe to mailbox
	subscribe chan *subscriptionRequest

	// Unsubscribe from mailbox
	unsubscribeMailbox chan *subscriptionRequest

	// Broadcast to mailbox subscribers
	broadcast chan *broadcastMessage

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Logger
	logger *slog.Logger
}

type subscriptionRequest struct {
	client    *Client
	mailboxID uint
}

type broadcastMessage struct {
	mailboxID uint
	message   []byte
}

// NewHub creates a new Hub instance
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:            make(map[*Client]bool),
		subscriptions:      make(map[uint]map[*Client]bool),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		subscribe:          make(chan *subscriptionRequest),
		unsubscribeMailbox: make(chan *subscriptionRequest),
		broadcast:          make(chan *broadcastMessage, 256),
		logger:             logger,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Debug("client registered")
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove from all subscriptions
				for mailboxID, subscribers := range h.subscriptions {
					delete(subscribers, client)
					if len(subscribers) == 0 {
						delete(h.subscriptions, mailboxID)
					}
				}
			}
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Debug("client unregistered")
			}

		case req := <-h.subscribe:
			h.mu.Lock()
			if h.subscriptions[req.mailboxID] == nil {
				h.subscriptions[req.mailboxID] = make(map[*Client]bool)
			}
			h.subscriptions[req.mailboxID][req.client] = true
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Debug("client subscribed to mailbox", slog.Uint64("mailbox_id", uint64(req.mailboxID)))
			}

		case req := <-h.unsubscribeMailbox:
			h.mu.Lock()
			if subscribers, ok := h.subscriptions[req.mailboxID]; ok {
				delete(subscribers, req.client)
				if len(subscribers) == 0 {
					delete(h.subscriptions, req.mailboxID)
				}
			}
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Debug("client unsubscribed from mailbox", slog.Uint64("mailbox_id", uint64(req.mailboxID)))
			}

		case msg := <-h.broadcast:
			h.mu.RLock()
			subscribers := h.subscriptions[msg.mailboxID]
			for client := range subscribers {
				select {
				case client.send <- msg.message:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Subscribe subscribes a client to a mailbox
func (h *Hub) Subscribe(client *Client, mailboxID uint) {
	h.subscribe <- &subscriptionRequest{client: client, mailboxID: mailboxID}
}

// Unsubscribe unsubscribes a client from a mailbox
func (h *Hub) Unsubscribe(client *Client, mailboxID uint) {
	h.unsubscribeMailbox <- &subscriptionRequest{client: client, mailboxID: mailboxID}
}

// BroadcastNewMessage broadcasts a new message notification to mailbox subscribers
func (h *Hub) BroadcastNewMessage(mailboxID uint, payload *NewMessagePayload) {
	msg := WSMessage{
		Type:      MessageTypeNewMessage,
		MailboxID: mailboxID,
		Message:   payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to marshal broadcast message", slog.Any("error", err))
		}
		return
	}

	h.broadcast <- &broadcastMessage{
		mailboxID: mailboxID,
		message:   data,
	}
}
