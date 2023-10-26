package block

import (
	"context"
	"fmt"
	"log"

	"cosmossdk.io/depinject"

	"pocket/pkg/client"
	"pocket/pkg/either"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
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
	// latestBlockObsvblsReplay is a replay observable with replay buffer size 1,
	// which holds the "active latest block observable" which is notified when
	// block commit events are received by the events query client subscription
	// created in goPublishBlocks. This observable (and the one it emits) closes
	// when the events bytes observable returns an error and is updated with a
	// new "active" observable after a new events query subscription is created.
	latestBlockObsvblsReplay observable.ReplayObservable[client.BlocksObservable]
	// latestBlockObsvblsReplayPublishCh is the publish channel for latestBlockObsvblsReplay.
	// It's used to set blockObsvbl initially and subsequently update it, for
	// example, when the connection is re-established after erroring.
	latestBlockObsvblsReplayPublishCh chan<- client.BlocksObservable
}

// eventsBytesToBlockMapFn is a convenience type to represent the type of a
// function which maps event subscription message bytes into block event objects.
// This is used as a transformFn in a channel.Map() call and is the type returned
// by the newEventsBytesToBlockMapFn factory function.
type eventBytesToBlockMapFn func(either.Either[[]byte]) (client.Block, bool)

func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (client.BlockClient, error) {
	// Initialize block client
	bClient := &blockClient{endpointURL: cometWebsocketURL}
	bClient.latestBlockObsvblsReplay, bClient.latestBlockObsvblsReplayPublishCh =
		channel.NewReplayObservable[client.BlocksObservable](ctx, 1)

	// Inject dependencies
	if err := depinject.Inject(deps, &bClient.eventsClient); err != nil {
		return nil, err
	}

	// Concurrently publish blocks to the observable emitted by latestBlockObsvblsReplay.
	go bClient.goPublishBlocks(ctx)

	return bClient, nil
}

// CommittedBlocksSequence returns a ReplayObservable, with a replay buffer size
// of 1, which is notified when block commit events are received by the events
// query subscription.
func (bClient *blockClient) CommittedBlocksSequence(ctx context.Context) client.BlocksObservable {
	replayedBlocksObservable := bClient.latestBlockObsvblsReplay.Last(ctx, 1)[0]
	return replayedBlocksObservable
}

// LatestBlock returns the latest committed block that's been received by the
// corresponding events query subscription.
// It blocks until at least one block event has been received.
func (bClient *blockClient) LatestBlock(ctx context.Context) (latestBlock client.Block) {
	v := bClient.CommittedBlocksSequence(ctx).Last(ctx, 1)[0]
	return v
}

// Close unsubscribes all observers of the committed blocks sequence observable
// and closes the events query client.
func (bClient *blockClient) Close() {
	// Closing eventsClient will cascade unsubscribe and close downstream observers.
	bClient.eventsClient.Close()
}

// goPublishBlocks receives event bytes from the events query client, maps them
// to block events, and publishes them to the latestBlockObsvblsReplay replay observable.
func (bClient *blockClient) goPublishBlocks(ctx context.Context) {
	// NB: cometbft event subscription query
	// (see: https://docs.cosmos.network/v0.47/core/events#subscribing-to-events)
	query := "tm.event='NewBlock'"

	// React to errors by getting a new events bytes observable, re-mapping it,
	// and send it to latestBlockObsvblsReplayPublishCh such that
	// latestBlockObsvblsReplay.Last(ctx, 1) will return it.
	retryOnError(ctx, "goPublishBlocks", func() chan error {
		errCh := make(chan error, 1)
		eventsBzObsvbl, err := bClient.eventsClient.EventsBytes(ctx, query)
		if err != nil {
			errCh <- err
			return errCh
		}

		// NB: must cast back to generic observable type to use with Map.
		// client.BlocksObservable is only used to workaround gomock's lack of
		// support for generic types.
		eventsBz := observable.Observable[either.Either[[]byte]](eventsBzObsvbl)
		blockEventFromEventBz := newEventsBytesToBlockMapFn(errCh)
		blocksObsvbl := channel.MapReplay(ctx, 1, eventsBz, blockEventFromEventBz)

		// Initially set latestBlockObsvblsReplay and update if after retrying on error.
		bClient.latestBlockObsvblsReplayPublishCh <- blocksObsvbl

		return errCh
	})
}

// retryOnError runs the given function, which is expected to return an error
// channel, and re-runs the function when an error is received.
// TODO_CONSIDERATION: promote to some shared package (perhaps /internal/concurrency)
func retryOnError(
	ctx context.Context,
	workName string,
	workFn func() chan error,
) {
	errCh := workFn()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			errCh = workFn()
			log.Printf("WARN: retrying %s after error: %s", workName, err)
		}
	}
}

// newEventsBytesToBlockMapFn is a factory for a function which is intended
// to be used as a transformFn in a channel.Map() call. Since the map function
// is called asynchronously, this factory creates a closure around an error channel
// which can be used for asynchronous error signaling from withing the map function,
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
