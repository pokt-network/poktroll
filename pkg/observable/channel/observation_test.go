package channel_test

import (
	"context"
	"sync"

	"github.com/pokt-network/poktroll/pkg/observable"
)

// NOTE: this file does not contain any tests, only test helpers.

// observation is a data structure that embeds an observer
// and keeps track of the received notifications.
// It uses generics with type parameter V.
type observation[V any] struct {
	// Embeds a mutex for thread-safe operations
	sync.Mutex
	// Embeds an Observer of type V
	observable.Observer[V]
	// Notifications is a slice of type V to store received notifications
	Notifications []V
}

// newObservation is a constructor function that returns
// a new observation instance. It subscribes to the provided observable.
func newObservation[V any](
	ctx context.Context,
	observable observable.Observable[V],
) *observation[V] {
	return &observation[V]{
		Observer:      observable.Subscribe(ctx),
		Notifications: []V{},
	}
}

// notify is a method on observation that safely
// appends a received value to the Notifications slice.
func (o *observation[V]) notify(value V) {
	o.Lock()         // Locks the mutex to prevent concurrent write access
	defer o.Unlock() // Unlocks the mutex when the method returns

	o.Notifications = append(o.Notifications, value) // Appends the received value to the Notifications slice
}

// goReceiveNotifications is a function that listens for
// notifications from the observer's channel and notifies
// the observation instance for each received value.
func goReceiveNotifications[V any](obsvn *observation[V]) {
	for notification := range obsvn.Ch() { // Listens for notifications on the channel
		obsvn.notify(notification) // Notifies the observation instance with the received value
	}
}
