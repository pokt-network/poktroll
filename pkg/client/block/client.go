package block

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/depinject"

	"pocket/pkg/client"
	"pocket/pkg/either"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
	"pocket/pkg/retry"
)

const (
	// eventsBytesRetryDelay is the delay between retry attempts when the events
	// bytes observable returns an error.
	eventsBytesRetryDelay = time.Second
	// eventsBytesRetryLimit is the maximum number of times to attempt to
	// re-establish the events query bytes subscription when the events bytes
	// observable returns an error.
	eventsBytesRetryLimit        = 10
	eventsBytesRetryResetTimeout = 10 * time.Second
	// NB: cometbft event subscription query for newly committed blocks.
	// (see: https://docs.cosmos.network/v0.47/core/events#subscribing-to-events)
	committedBlocksQuery = "tm.event='NewBlock'"
	// latestBlockObsvblsReplayBufferSize is the replay buffer size of the
	// latestBlockObsvbls replay observable which is used to cache the latest block observable.
	// It is updated with a new "active" observable when a new
	// events query subscription is created, for example, after a non-persistent
	// connection error.
	latestBlockObsvblsReplayBufferSize = 1
	// latestBlockReplayBufferSize is the replay buffer size of the latest block
	// replay observable which is notified when block commit events are received
	// by the events query client subscription created in goPublishBlocks.
	latestBlockReplayBufferSize = 1
)

var (
	_ client.BlockClient = (*blockClient)(nil)
	_ client.Block       = (*cometBlockEvent)(nil)
)

// blockClient implements the BlockClient interface.
type blockClient struct {
	// endpointURL is the URL of RPC endpoint which eventsClient subscription
	// requests will be sent.
	endpointURL string
	// eventsClient is the events query client which is used to subscribe to
	// newly committed block events. It emits an either value which may contain
	// an error, at most, once and closes immediately after if it does.
	eventsClient client.EventsQueryClient
	// latestBlockObsvbls is a replay observable with replay buffer size 1,
	// which holds the "active latest block observable" which is notified when
	// block commit events are received by the events query client subscription
	// created in goPublishBlocks. This observable (and the one it emits) closes
	// when the events bytes observable returns an error and is updated with a
	// new "active" observable after a new events query subscription is created.
	latestBlockObsvbls observable.ReplayObservable[client.BlocksObservable]
	// latestBlockObsvblsReplayPublishCh is the publish channel for latestBlockObsvbls.
	// It's used to set blockObsvbl initially and subsequently update it, for
	// example, when the connection is re-established after erroring.
	latestBlockObsvblsReplayPublishCh chan<- client.BlocksObservable
}

// eventsBytesToBlockMapFn is a convenience type to represent the type of a
// function which maps event subscription message bytes into block event objects.
// This is used as a transformFn in a channel.Map() call and is the type returned
// by the newEventsBytesToBlockMapFn factory function.
type eventBytesToBlockMapFn func(either.Either[[]byte]) (client.Block, bool)

// NewBlockClient creates a new block client from the given dependencies and cometWebsocketURL.
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (client.BlockClient, error) {
	// Initialize block client
	bClient := &blockClient{endpointURL: cometWebsocketURL}
	bClient.latestBlockObsvbls, bClient.latestBlockObsvblsReplayPublishCh =
		channel.NewReplayObservable[client.BlocksObservable](ctx, latestBlockObsvblsReplayBufferSize)

	// Inject dependencies
	if err := depinject.Inject(deps, &bClient.eventsClient); err != nil {
		return nil, err
	}

	// Concurrently publish blocks to the observable emitted by latestBlockObsvbls.
	go bClient.goPublishBlocks(ctx)

	return bClient, nil
}

// CommittedBlocksSequence returns a ReplayObservable, with a replay buffer size
// of 1, which is notified when block commit events are received by the events
// query subscription.
func (bClient *blockClient) CommittedBlocksSequence(ctx context.Context) client.BlocksObservable {
	// Get the latest block observable from the replay observable. We only ever
	// want the last 1 as any prior latest block observable values are closed.
	// Directly accessing the zeroth index here is safe because the call to Last
	// is guaranteed to return a slice with at least 1 element.
	return bClient.latestBlockObsvbls.Last(ctx, 1)[0]
}

// LatestBlock returns the latest committed block that's been received by the
// corresponding events query subscription.
// It blocks until at least one block event has been received.
func (bClient *blockClient) LatestBlock(ctx context.Context) client.Block {
	return bClient.CommittedBlocksSequence(ctx).Last(ctx, 1)[0]
}

// Close unsubscribes all observers of the committed blocks sequence observable
// and closes the events query client.
func (bClient *blockClient) Close() {
	// Closing eventsClient will cascade unsubscribe and close downstream observers.
	bClient.eventsClient.Close()
}

// goPublishBlocks runs the work function returned by retryPublishBlocksFactory,
// re-invoking it according to the arguments to retry.OnError when the events bytes
// observable returns an asynchronous error.
// This function is intended to be called in a goroutine.
func (bClient *blockClient) goPublishBlocks(ctx context.Context) {
	// React to errors by getting a new events bytes observable, re-mapping it,
	// and send it to latestBlockObsvblsReplayPublishCh such that
	// latestBlockObsvbls.Last(ctx, 1) will return it.
	publishErr := retry.OnError(
		ctx,
		eventsBytesRetryLimit,
		eventsBytesRetryDelay,
		eventsBytesRetryResetTimeout,
		"goPublishBlocks",
		bClient.retryPublishBlocksFactory(ctx),
	)

	// If we get here, the retry limit was reached and the retry loop exited.
	// Since this function runs in a goroutine, we can't return the error to the
	// caller. Instead, we panic.
	panic(fmt.Errorf("BlockClient.goPublishBlocks shold never reach this spot: %w", publishErr))
}

// retryPublishBlocksFactory returns a function which is intended to be passed to
// retry.OnError. The returned function pipes event bytes from the events query
// client, maps them to block events, and publishes them to the latestBlockObsvbls
// replay observable.
func (bClient *blockClient) retryPublishBlocksFactory(ctx context.Context) func() chan error {
	return func() chan error {
		errCh := make(chan error, 1)
		eventsBzObsvbl, err := bClient.eventsClient.EventsBytes(ctx, committedBlocksQuery)
		if err != nil {
			errCh <- err
			return errCh
		}

		// NB: must cast back to generic observable type to use with Map.
		// client.BlocksObservable is only used to workaround gomock's lack of
		// support for generic types.
		eventsBz := observable.Observable[either.Either[[]byte]](eventsBzObsvbl)
		blockEventFromEventBz := newEventsBytesToBlockMapFn(errCh)
		blocksObsvbl := channel.MapReplay(ctx, latestBlockReplayBufferSize, eventsBz, blockEventFromEventBz)

		// Initially set latestBlockObsvbls and update if after retrying on error.
		bClient.latestBlockObsvblsReplayPublishCh <- blocksObsvbl

		return errCh
	}
}

// newEventsBytesToBlockMapFn is a factory for a function which is intended
// to be used as a transformFn in a channel.Map() call. Since the map function
// is called asynchronously, this factory creates a closure around an error channel
// which can be used for asynchronous error signaling from within the map function,
// and handling from the Map call context.
//
// The map function itself attempts to deserialize the given byte slice as a
// committed block event. If the events bytes observable contained an error, this value is not emitted
// (skipped) on the destination observable of the map operation.
// If deserialization failed because the event bytes were for a different event type,
// this value is also skipped.
// If deserialization failed for some other reason, this function panics.
func newEventsBytesToBlockMapFn(errCh chan<- error) eventBytesToBlockMapFn {
	return func(eitherEventBz either.Either[[]byte]) (_ client.Block, skip bool) {
		eventBz, err := eitherEventBz.ValueOrError()
		if err != nil {
			errCh <- err
			// Don't publish (skip) if eitherEventBz contained an error.
			// eitherEventBz should automatically close itself in this case.
			// (i.e. no more values should be mapped to this transformFn's respective
			// dstObservable).
			return nil, true
		}

		block, err := newCometBlockEvent(eventBz)
		if err != nil {
			if ErrUnmarshalBlockEvent.Is(err) {
				// Don't publish (skip) if the message was not a block event.
				return nil, true
			}

			panic(fmt.Sprintf(
				"unexpected error deserializing block event: %s; eventBz: %s",
				err, string(eventBz),
			))
		}
		return block, false
	}
}
