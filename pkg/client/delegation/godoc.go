// Package delegation contains a light wrapper of the EventsReplayClient[Redelegation]
// generic which listens for redelegation events on chain and emits them
// through a ReplayObservable. This enables consumers to listen for onchain
// application redelegation events and react to them asynchronously.
package delegation
