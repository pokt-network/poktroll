package events

import "github.com/pokt-network/poktroll/pkg/client"

// WithDialer returns a client.EventsQueryClientOption which sets the given dialer on the
// resulting eventsQueryClient when passed to NewEventsQueryClient().
func WithDialer(dialer client.Dialer) client.EventsQueryClientOption {
	return func(evtClient client.EventsQueryClient) {
		evtClient.(*eventsQueryClient).dialer = dialer
	}
}

// WithConnRetryLimit returns an option function which sets the number
// of times the replay client should retry in the event that it encounters
// an error or its connection is interrupted.
// If connRetryLimit is < 0, it will retry indefinitely.
func WithConnRetryLimit[T any](limit int) client.EventsReplayClientOption[T] {
	return func(client client.EventsReplayClient[T]) {
		client.(*replayClient[T]).connRetryLimit = limit
	}
}
