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
//
// For production environments, it's recommended to use a negative value
// (e.g., -1) to enable unlimited retries with exponential backoff.
// Example usage:
//
//	client, err := events.NewEventsReplayClient(
//		ctx,
//		deps,
//		queryString,
//		newEventFn,
//		replayObsBufferSize,
//		events.WithConnRetryLimit[MyEventType](-1), // Unlimited retries
//	)
func WithConnRetryLimit[T any](limit int) client.EventsReplayClientOption[T] {
	return func(client client.EventsReplayClient[T]) {
		// Ignore the zero value because limit may be provided via a partially
		// configured config struct (i.e. no retry limit set).
		// The default will be used instead.
		if limit != 0 {
			client.(*replayClient[T]).connRetryLimit = limit
		}
	}
}
