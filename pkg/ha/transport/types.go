package transport

import (
	"time"
)

// SetPublishedAt sets the PublishedAtUnixNano field to the current time.
func (m *MinedRelayMessage) SetPublishedAt() {
	m.PublishedAtUnixNano = time.Now().UnixNano()
}

// PublishedAt returns the PublishedAtUnixNano as a time.Time.
func (m *MinedRelayMessage) PublishedAt() time.Time {
	return time.Unix(0, m.PublishedAtUnixNano)
}

// StreamMessage wraps a MinedRelayMessage with its Redis Stream message ID.
// Used by the consumer to track which messages have been processed.
type StreamMessage struct {
	// ID is the Redis Stream message ID (e.g., "1234567890123-0").
	ID string

	// Message is the deserialized MinedRelayMessage.
	Message *MinedRelayMessage
}
