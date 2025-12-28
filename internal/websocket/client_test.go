package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_CreatesClientWithConnection(t *testing.T) {
	hub := NewHub(nil)

	// We can't easily create a real websocket.Conn in tests,
	// but we can test that NewClient returns a properly initialized client
	client := NewClient(hub, nil, nil)

	assert.NotNil(t, client)
	assert.Equal(t, hub, client.hub)
	assert.NotNil(t, client.send)
}

func TestClient_HandleMessage_ProcessesSubscribe(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()

	client := NewClient(hub, nil, nil)

	// Register client first
	hub.Register(client)
	time.Sleep(10 * time.Millisecond) // Allow registration to process

	// Create subscribe message
	msg := WSMessage{
		Type:      MessageTypeSubscribe,
		MailboxID: 123,
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Handle the message
	client.handleMessage(data)
	time.Sleep(10 * time.Millisecond) // Allow subscription to process

	// Verify subscription was added
	hub.mu.RLock()
	_, exists := hub.subscriptions[123]
	hub.mu.RUnlock()

	assert.True(t, exists)
}

func TestClient_HandleMessage_ProcessesUnsubscribe(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()

	client := NewClient(hub, nil, nil)

	// Register and subscribe client
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	hub.Subscribe(client, 123)
	time.Sleep(10 * time.Millisecond)

	// Create unsubscribe message
	msg := WSMessage{
		Type:      MessageTypeUnsubscribe,
		MailboxID: 123,
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	// Handle the message
	client.handleMessage(data)
	time.Sleep(10 * time.Millisecond)

	// Verify subscription was removed
	hub.mu.RLock()
	subscribers, exists := hub.subscriptions[123]
	hub.mu.RUnlock()

	// Either the subscription doesn't exist or the client is not in it
	if exists {
		_, clientExists := subscribers[client]
		assert.False(t, clientExists)
	}
}

func TestClient_HandleMessage_SendsErrorForInvalidJSON(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient(hub, nil, nil)

	// Send invalid JSON
	client.handleMessage([]byte("invalid json"))

	// Check that error was sent
	select {
	case msg := <-client.send:
		var wsMsg WSMessage
		err := json.Unmarshal(msg, &wsMsg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeError, wsMsg.Type)
		assert.Contains(t, wsMsg.Error, "invalid message format")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected error message to be sent")
	}
}

func TestClient_HandleMessage_SendsErrorForUnknownType(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient(hub, nil, nil)

	// Send message with unknown type
	msg := WSMessage{
		Type: "unknown_type",
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	client.handleMessage(data)

	// Check that error was sent
	select {
	case msg := <-client.send:
		var wsMsg WSMessage
		err := json.Unmarshal(msg, &wsMsg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeError, wsMsg.Type)
		assert.Contains(t, wsMsg.Error, "unknown message type")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected error message to be sent")
	}
}

func TestClient_HandleMessage_SendsErrorForMissingMailboxID(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient(hub, nil, nil)

	// Send subscribe without mailbox_id
	msg := WSMessage{
		Type:      MessageTypeSubscribe,
		MailboxID: 0,
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	client.handleMessage(data)

	// Check that error was sent
	select {
	case msg := <-client.send:
		var wsMsg WSMessage
		err := json.Unmarshal(msg, &wsMsg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeError, wsMsg.Type)
		assert.Contains(t, wsMsg.Error, "mailbox_id is required")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected error message to be sent")
	}
}

func TestClient_SendError_SendsErrorMessage(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient(hub, nil, nil)

	client.sendError("test error")

	// Check that error was sent
	select {
	case msg := <-client.send:
		var wsMsg WSMessage
		err := json.Unmarshal(msg, &wsMsg)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeError, wsMsg.Type)
		assert.Equal(t, "test error", wsMsg.Error)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected error message to be sent")
	}
}

func TestWSMessage_JSONSerialization(t *testing.T) {
	msg := WSMessage{
		Type:      MessageTypeNewMessage,
		MailboxID: 123,
		Message: map[string]interface{}{
			"id":      1,
			"subject": "Test",
		},
	}

	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded WSMessage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, MessageTypeNewMessage, decoded.Type)
	assert.Equal(t, uint(123), decoded.MailboxID)
}

func TestNewMessagePayload_JSONSerialization(t *testing.T) {
	payload := NewMessagePayload{
		ID:          1,
		SenderEmail: "test@example.com",
		SenderName:  "Test User",
		Subject:     "Test Subject",
		ReceivedAt:  "2025-01-01T00:00:00Z",
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var decoded NewMessagePayload
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, uint(1), decoded.ID)
	assert.Equal(t, "test@example.com", decoded.SenderEmail)
	assert.Equal(t, "Test User", decoded.SenderName)
	assert.Equal(t, "Test Subject", decoded.Subject)
}

func TestMessageTypes_AreCorrectValues(t *testing.T) {
	assert.Equal(t, MessageType("subscribe"), MessageTypeSubscribe)
	assert.Equal(t, MessageType("unsubscribe"), MessageTypeUnsubscribe)
	assert.Equal(t, MessageType("new_message"), MessageTypeNewMessage)
	assert.Equal(t, MessageType("error"), MessageTypeError)
}

func TestClient_SendChannel_HasBuffer(t *testing.T) {
	hub := NewHub(nil)
	client := NewClient(hub, nil, nil)

	// Should be able to send multiple messages without blocking
	for i := 0; i < 10; i++ {
		client.sendError("test error")
	}

	// Verify messages were buffered
	count := 0
	for {
		select {
		case <-client.send:
			count++
		default:
			goto done
		}
	}
done:
	assert.Equal(t, 10, count)
}
