package eventsquery

import "pocket/pkg/client"

// WithDialer returns a client.EventsQueryClientOption which sets the given dialer on the
// resulting eventsQueryClient when passed to NewEventsQueryClient().
func WithDialer(dialer client.Dialer) client.EventsQueryClientOption {
	return func(evtClient client.EventsQueryClient) {
		evtClient.(*eventsQueryClient).dialer = dialer
	}
}
