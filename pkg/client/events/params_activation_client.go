package events

import (
	"context"
	"strconv"

	"cosmossdk.io/depinject"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/json"
	cmtcoretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	comettypes "github.com/cometbft/cometbft/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

// paramsActivatedQuery defines the Tendermint websocket query used to subscribe to
// EventParamsActivated events emitted by the chain.
// It listens for any occurrence of this event.
const paramsActivatedQuery = "pocket.EventParamsActivated EXISTS"

var _ client.EventsParamsActivationClient = (*eventsParamsActivationClient)(nil)

// eventsParamsActivationClient implements the EventsParamsActivationClient interface.
// It provides functionality to subscribe to parameter update events from the chain
// via websocket connection and exposes them as an observable stream for queriers and other
// components to consume.
type eventsParamsActivationClient struct {
	// latestParamsUpdateObs is an observable that streams emits proto.Message events
	// whenever parameters are updated on the chain.
	latestParamsUpdateObs observable.Observable[proto.Message]
}

// NewEventsParamsActivationClient creates and initializes a new EventsParamsActivationClient.
//
// It sets up a subscription to parameter activation events from the chain via websocket,
// processes these events, and provides an observable stream for other components to consume
// parameter updates.
//
// Required dependencies:
//   - client.EventsQueryClient: For subscribing to chain events
func NewEventsParamsActivationClient(
	ctx context.Context,
	deps depinject.Config,
) (_ client.EventsParamsActivationClient, err error) {
	// Create a new observable channel for parameter update events
	latestParamsUpdateObs, latestParamsUpdateCh := channel.NewObservable[proto.Message]()
	epaClient := &eventsParamsActivationClient{latestParamsUpdateObs: latestParamsUpdateObs}

	var eventsQueryClient client.EventsQueryClient
	if err := depinject.Inject(deps, &eventsQueryClient); err != nil {
		return nil, err
	}

	// Subscribe to parameter activation events using EventsBytes which provides
	// a websocket subscription to events matching the query
	eventsBytesObs, err := eventsQueryClient.EventsBytes(ctx, paramsActivatedQuery)
	if err != nil {
		return nil, err
	}

	// Start asynchronously forwarding parameter update events to our observable channel
	go epaClient.asyncForwardParamsUpdate(ctx, eventsBytesObs, latestParamsUpdateCh)

	return epaClient, nil
}

// LatestParamsUpdate returns an observable stream that emits parameter update events
// whenever parameters are updated on the chain.
//
// Subscribers to this observable will receive parameter updates in real-time as they
// occur on the chain, which enables components like queriers to maintain up-to-date
// parameter caches.
func (epaClient *eventsParamsActivationClient) LatestParamsUpdate() observable.Observable[proto.Message] {
	return epaClient.latestParamsUpdateObs
}

// asyncForwardParamsUpdate sets up an asynchronous processing pipeline that:
// 1. Listens for raw event bytes from the EventsBytes observable
// 2. Processes these bytes to extract parameter activation events
// 3. Forwards the extracted events to the provided channel
//
// This method is called during client initialization and runs for the lifetime of the
// provided context. It handles the transformation of raw websocket data into typed
// parameter update events that can be consumed by other components.
func (epaClient *eventsParamsActivationClient) asyncForwardParamsUpdate(
	ctx context.Context,
	eventsBytesObs client.EventsBytesObservable,
	latestParamsUpdatePublishCh chan<- proto.Message,
) {
	channel.ForEach(
		ctx,
		eventsBytesObs,
		func(ctx context.Context, eventsBytes either.Bytes) {
			// Extract the raw bytes from the either.Bytes container
			// Return early if an error occurred in the observable
			blockMsgBz, err := eventsBytes.ValueOrError()
			if err != nil {
				return
			}

			// Parse the raw bytes into an RPC response
			var rpcResponse rpctypes.RPCResponse
			if err := json.Unmarshal(blockMsgBz, &rpcResponse); err != nil {
				return
			}

			// Extract the block results from the RPC response
			var resultEvent cmtcoretypes.ResultEvent
			err = json.Unmarshal(rpcResponse.Result, &resultEvent)
			if err != nil {
				return
			}

			if resultEvent.Data == nil {
				return
			}

			eventNewBlockEvents, ok := resultEvent.Data.(comettypes.EventDataNewBlock)
			if !ok {
				return
			}

			// Process each event in the block's FinalizeBlock events
			for _, event := range eventNewBlockEvents.ResultFinalizeBlock.Events {
				// Quote the 'mode' attribute in the event to avoid issues with parsing
				QuoteEventMode(&event)
				// Parse the event into a typed event using Cosmos SDK utilities
				typedEvent, err := cosmostypes.ParseTypedEvent(event)
				if err != nil {
					continue
				}

				// Forward the typed event to subscribers via the channel
				latestParamsUpdatePublishCh <- typedEvent
			}
		},
	)
}

func QuoteEventMode(event *abci.Event) {
	for i, attr := range event.Attributes {
		if attr.Key == "mode" {
			event.Attributes[i].Value = strconv.Quote(attr.Value)
			return
		}
	}
}
