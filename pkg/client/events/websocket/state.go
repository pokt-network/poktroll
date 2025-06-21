package websocket

import (
	"time"
)

// ConnStateStatus represents the status of the underlying websocket connection
type ConnStateStatus int

const (
	// ConnStateInitial represents the initial state of the underlying websocket connection
	ConnStateInitial ConnStateStatus = iota
	// ConnStateConnected represents a connected state
	ConnStateConnected
	// ConnStateDisconnected represents a disconnected state
	ConnStateDisconnected
	// ConnStateWaitingRetry represents a state where the client is waiting to retry connection
	ConnStateWaitingRetry
	// ConnStateFailed represents a failed connection state
	ConnStateFailed
	// ConnStateDecodeError represents a state where there was an error decoding events
	ConnStateDecodeError
)

// String returns the string representation of ConnStateStatus
func (s ConnStateStatus) String() string {
	return [...]string{
		"initial",
		"connected",
		"disconnected",
		"waiting_retry",
		"failed",
		"decode_error",
	}[s]
}

// Define connection state for tracking and logging transitions
type ConnState struct {
	Status    ConnStateStatus
	Timestamp time.Time
}
