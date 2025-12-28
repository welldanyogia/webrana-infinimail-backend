package mocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/websocket"
)

// NotificationRecord records a notification sent through the mock hub
type NotificationRecord struct {
	MailboxID uint
	Payload   *websocket.NewMessagePayload
}

// MockWebSocketHub implements a mock for websocket.Hub
type MockWebSocketHub struct {
	mock.Mock
	Notifications []NotificationRecord
}

// NewMockWebSocketHub creates a new MockWebSocketHub instance
func NewMockWebSocketHub() *MockWebSocketHub {
	return &MockWebSocketHub{
		Notifications: make([]NotificationRecord, 0),
	}
}

// Run starts the hub's main loop (no-op for mock)
func (m *MockWebSocketHub) Run() {
	m.Called()
}

// Register adds a client to the hub
func (m *MockWebSocketHub) Register(client *websocket.Client) {
	m.Called(client)
}

// Unregister removes a client from the hub
func (m *MockWebSocketHub) Unregister(client *websocket.Client) {
	m.Called(client)
}

// Subscribe subscribes a client to a mailbox
func (m *MockWebSocketHub) Subscribe(client *websocket.Client, mailboxID uint) {
	m.Called(client, mailboxID)
}

// Unsubscribe unsubscribes a client from a mailbox
func (m *MockWebSocketHub) Unsubscribe(client *websocket.Client, mailboxID uint) {
	m.Called(client, mailboxID)
}

// BroadcastNewMessage broadcasts a new message notification to mailbox subscribers
func (m *MockWebSocketHub) BroadcastNewMessage(mailboxID uint, payload *websocket.NewMessagePayload) {
	m.Called(mailboxID, payload)
	m.Notifications = append(m.Notifications, NotificationRecord{
		MailboxID: mailboxID,
		Payload:   payload,
	})
}

// GetNotifications returns all recorded notifications
func (m *MockWebSocketHub) GetNotifications() []NotificationRecord {
	return m.Notifications
}

// ClearNotifications clears all recorded notifications
func (m *MockWebSocketHub) ClearNotifications() {
	m.Notifications = make([]NotificationRecord, 0)
}
