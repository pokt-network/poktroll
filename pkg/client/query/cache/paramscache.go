package cache

import (
	"context"
	"time"

	proto "github.com/cosmos/gogoproto/proto"
	grpc "google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/retry"
)

const (
	deadlineSeconds = 5
)

var _ client.ParamsCache[any] = (*paramsCache[any])(nil)

// singleValueCache is the key used to store the value in the cache.
const singleValueCache = ""

// paramsCache is a simple in-memory historical cache implementation for query parameters.
// It does not involve key-value pairs, but only stores a single value.
type paramsCache[T any] struct {
	historicalKeyValueCache cache.HistoricalKeyValueCache[T]
}

// NewParamsCache returns a new instance of a ParamsCache.
func NewParamsCache[T any](opts ...memory.KeyValueCacheOptionFn) (*paramsCache[T], error) {
	historicalKeyValueCache, err := memory.NewHistoricalKeyValueCache[T](opts...)
	if err != nil {
		return nil, err
	}

	return &paramsCache[T]{
		historicalKeyValueCache,
	}, nil
}

// GetLatest returns the latest value stored in the cache.
// A boolean is returned as the second value to indicate if the value was found in the cache.
func (c *paramsCache[T]) GetLatest() (value T, found bool) {
	return c.historicalKeyValueCache.GetLatestVersion(singleValueCache)
}

// GetAtHeight returns the value stored in the cache at the given height.
func (c *paramsCache[T]) GetAtHeight(height int64) (value T, found bool) {
	return c.historicalKeyValueCache.GetVersionLTE(singleValueCache, height)
}

// Set stores a value in the cache at the given height.
func (c *paramsCache[T]) SetAtHeight(value T, height int64) {
	c.historicalKeyValueCache.SetVersion(singleValueCache, value, height)
}

// Get all versions of a value stored in the cache.
func (c *paramsCache[T]) GetAllUpdates() (cache.CacheValueHistory[T], bool) {
	return c.historicalKeyValueCache.GetAllVersions(singleValueCache)
}

// paramsUpdate is an interface for types that provide access to parameters and their activation height.
// It allows the cache system to store parameters at specific blockchain heights.
type paramsUpdate[P any] interface {
	// GetParams returns the parameters of type P.
	GetParams() P
	// GetActivationHeight returns the block height at which these parameters become active.
	GetActivationHeight() int64
}

// paramsUpdateResponse is an interface for response types that contain the parameter updates history.
// This is typically returned from gRPC query services.
type paramsUpdateResponse[U any] interface {
	// GetParamsUpdates returns the parameters updates history.
	GetParamsUpdates() U
}

// eventParamsActivated is an interface for event types that signal parameter activation.
// These events are emitted when parameters are activated at their respective heights.
type eventParamsActivated interface {
	// GetParamsUpdate returns the parameter update that has been activated.
	GetParamsUpdate() any
}

// paramsQuerier is an interface for components that can query parameter updates from a service.
// This typically wraps a gRPC client that fetches parameter updates.
type paramsQuerier[Req any, Res any] interface {
	// ParamsUpdates queries for parameter updates using the provided request.
	ParamsUpdates(ctx context.Context, req Req, opts ...grpc.CallOption) (Res, error)
}

// UpdateParamsCache populates and maintains a parameter cache by fetching historical updates
// and subscribing to live parameter update events.
//
// Generic Parameters:
// P - The parameter type that will be stored in the cache (e.g., application.Params)
// U - A type implementing paramsUpdate[P] interface that provides access to parameters and their activation height
// Req - The request type used when querying for parameter updates (e.g., QueryParamsUpdatesRequest)
// Res - The response type from parameter update queries, implementing paramsUpdateResponse[[]U]
//
// The function performs two main operations:
// 1. Subscribes to live parameter update events and caches them as they occur
// 2. Fetches the historical parameter updates to populate the cache with past values
func UpdateParamsCache[
	P any,
	U paramsUpdate[P],
	Req any,
	Res paramsUpdateResponse[[]U],
](
	ctx context.Context,
	req Req,
	toParamsUpdate func(protoMessage proto.Message) (U, bool),
	querier paramsQuerier[Req, Res],
	paramsUpdatesClient client.EventsParamsActivationClient,
	cache client.ParamsCache[P],
) error {
	// Subscribe to the parameters updates observable:
	// - Iterate over the stream of parameter updates provided by `ParamsUpdatesClient.LatestParamsUpdate(ctx)`.
	// - For each observed update, check if the message is of type `*apptypes.EventParamsActivated`.
	// - It extracts the matching `ParamsUpdate` and their activation height from the event.
	// - These parameters are then cached, ensuring they are stored at their respective activation heights.
	//
	// DEV_NOTE: Why this can run in parallel:
	// - The caching mechanism (`SetAtHeight`) ensures that updates are stored at their specific activation heights.
	// - If a value is overridden at a height, it is guaranteed to be the same value.
	// - This eliminates the risk of race conditions or inconsistent data, making the operation safe to run in parallel.
	channel.ForEach(
		ctx,
		paramsUpdatesClient.LatestParamsUpdate(),
		func(ctx context.Context, protoMessage proto.Message) {
			paramsUpdate, ok := toParamsUpdate(protoMessage)
			if !ok {
				return
			}

			cache.SetAtHeight(paramsUpdate.GetParams(), paramsUpdate.GetActivationHeight())
		},
	)

	// Set a deadline for the context to avoid long-running initialization.
	// This is a safeguard to ensure that the initialization does not block indefinitely.
	ctxWithDeadline, cancel := context.WithDeadline(ctx, time.Now().Add(deadlineSeconds*time.Second))
	defer cancel()

	// Fetch the parameter update history from the querier.
	paramsHistory, err := retry.Call(ctx, func() (Res, error) {
		return querier.ParamsUpdates(ctxWithDeadline, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return err
	}

	// Populate the cache with the parameter updates and their activation heights.
	for _, paramUpdate := range paramsHistory.GetParamsUpdates() {
		// Cache each parameter update at its respective activation height.
		cache.SetAtHeight(paramUpdate.GetParams(), paramUpdate.GetActivationHeight())
	}
	return nil
}
