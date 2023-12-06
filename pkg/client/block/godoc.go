// Package block contains a light wrapper of the EventsReplayClient[Block]
// generic which listens for committed block events on chain and emits them
// through a ReplayObservable. This enables consumers to listen for newly
// committed blocks and react to them asynchronously.
package block
