// Package events provides a generic client for subscribing to onchain events
// via an EventsQueryClient and transforming the received events into the type
// defined by the EventsReplayClient's generic type parameter.
//
// The EventsQueryClient emits ReplayObservables which are of the type defined
// by the EventsReplayClient's generic type parameter.
//
// The usage of of ReplayObservables allows the EventsReplayClient to be always
// provide the latest event data to the caller, even if the connection to the
// EventsQueryClient is lost and re-established, without the caller having to
// re-subscribe to the EventsQueryClient.
package events
