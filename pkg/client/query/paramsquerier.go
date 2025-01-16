package query

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cosmossdk.io/depinject"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

// abstractParamsQuerier is NOT intended to be used for anything except the
// compile-time interface compliance assertion that immediately follows.
type abstractParamsQuerier = cachedParamsQuerier[cosmostypes.Msg, paramsQuerierIface[cosmostypes.Msg]]

var _ client.ParamsQuerier[cosmostypes.Msg] = (*abstractParamsQuerier)(nil)

// paramsQuerierIface is an interface which generated query clients MUST implement
// to be compatible with the cachedParamsQuerier.
//
// DEV_NOTE: It is mainly required due to syntactic constraints imposed by the generics
// (i.e. otherwise, P here MUST be a value type, and there's no way to express that Q
// (below) SHOULD be in terms of the concrete type of P in NewCachedParamsQuerier).
type paramsQuerierIface[P cosmostypes.Msg] interface {
	GetParams(context.Context) (P, error)
}

// NewCachedParamsQuerier creates a new, generic, params querier with the given
// concrete query client constructor and the configuration which results from
// applying the given options.
func NewCachedParamsQuerier[P cosmostypes.Msg, Q paramsQuerierIface[P]](
	ctx context.Context,
	deps depinject.Config,
	queryClientConstructor func(conn gogogrpc.ClientConn) Q,
	opts ...ParamsQuerierOptionFn,
) (_ client.ParamsQuerier[P], err error) {
	cfg := DefaultParamsQuerierConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err = cfg.Validate(); err != nil {
		return nil, err
	}

	//paramsCache, err := cache.NewInMemoryCache[P](cfg.cacheOpts...)
	//if err != nil {
	//	return nil, err
	//}

	querier := &cachedParamsQuerier[P, Q]{
		config: cfg,
		//paramsCache: paramsCache,
	}

	if err = depinject.Inject(
		deps,
		&querier.clientConn,
		&querier.paramsCache,
		&querier.blockClient,
	); err != nil {
		return nil, err
	}

	// Construct the module-specific query client.
	querier.queryClient = queryClientConstructor(querier.clientConn)

	// Construct an events replay client which is notified about txs which were
	// signed by the governance module account; this includes all parameter update
	// messages for all modules while excluding almost all other events, reducing
	// bandwidth utilization.
	query := fmt.Sprintf(govAccountTxQueryFmt, authtypes.NewModuleAddress(govtypes.ModuleName))
	querier.eventsReplayClient, err = events.NewEventsReplayClient(ctx, deps, query, tx.UnmarshalTxResult, 1)
	if err != nil {
		return
	}

	// Prime the cache by querying for the current params.
	if _, err = querier.GetParams(ctx); err != nil {
		return nil, err
	}

	// Subscribe to asynchronous events to keep the cache up-to-date.
	go querier.goSubscribeToParamUpdates(ctx)

	return querier, nil
}

// cachedParamsQuerier provides a generic implementation of cached param querying.
// It handles parameter caching and chain querying in a generic way, where
// P is a pointer type of the parameters, and Q is the interface type of the
// corresponding query client.
type cachedParamsQuerier[P cosmostypes.Msg, Q paramsQuerierIface[P]] struct {
	clientConn         gogogrpc.ClientConn
	queryClient        Q
	eventsReplayClient client.EventsReplayClient[*abcitypes.TxResult]
	blockClient        client.BlockClient
	paramsCache        client.HistoricalQueryCache[P]
	config             *paramsQuerierConfig
}

// GetParams returns the latest cached params, if any; otherwise, it queries the
// current on-chain params and caches them.
func (bq *cachedParamsQuerier[P, Q]) GetParams(ctx context.Context) (P, error) {
	logger := bq.config.logger.With(
		"method", "GetParams",
	)

	// Check the cache first.
	var paramsZero P
	cached, err := bq.paramsCache.Get("params")
	switch {
	case err == nil:
		logger.Debug().Msgf("params cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return paramsZero, err
	}

	logger.Debug().Msgf("%s", err)

	// Query on-chain on cache miss.
	params, err := bq.queryClient.GetParams(ctx)
	if err != nil {
		if bq.config.moduleParamError != nil {
			return paramsZero, bq.config.moduleParamError.Wrap(err.Error())
		}
		return paramsZero, err
	}

	return params, nil
}

// GetParamsAtHeight returns parameters as they were as of the given height, **if
// that height is present in the cache**. Otherwise, it queries the current params
// and returns them.
//
// TODO_MAINNET(@bryanchriswhite): Once on-chain historical data is available,
// update this to query for the historical params, rather than returning the
// current params, if the case of a cache miss.
func (bq *cachedParamsQuerier[P, Q]) GetParamsAtHeight(ctx context.Context, height int64) (P, error) {
	logger := bq.config.logger.With(
		"method", "GetParamsAtHeight",
		"height", height,
	)

	// Try to get from cache at specific height
	cached, err := bq.paramsCache.GetAsOfVersion("params", height)
	switch {
	case err == nil:
		logger.Debug().Msg("params cache hit")
		return cached, nil
	case !errors.Is(err, cache.ErrCacheMiss):
		return cached, err
	}

	logger.Debug().Msgf("%s", err)

	// TODO_MAINNET(@bryanchriswhite): Implement querying historical params from chain
	err = cache.ErrCacheMiss.Wrapf("TODO: on-chain historical data not implemented")
	logger.Error().Msgf("%s", err)

	// Meanwhile, return current params as fallback. 😬
	return bq.GetParams(ctx)
}

// TODO_IN_THIS_COMMIT: godoc...
var govAccountTxQueryFmt = "tm.event='Tx' AND message.sender='%s'"

// TODO_IN_THIS_COMMIT: godoc...
func (bq *cachedParamsQuerier[P, Q]) goSubscribeToParamUpdates(ctx context.Context) {
	govSignedTxResultsObs := bq.eventsReplayClient.EventsSequence(ctx)
	channel.ForEach[*abcitypes.TxResult](
		ctx, govSignedTxResultsObs,
		func(ctx context.Context, txEvent *abcitypes.TxResult) {
			txEventJSON, err := json.MarshalIndent(txEvent, "", "  ")
			if err != nil {
				panic(err)
			}
			fmt.Printf(">>> event: %s\n", txEventJSON)
		},
	)

	//govSignedTxResultsCh := govSignedTxResultsObs.Subscribe(ctx).Ch()

	//for govSignedTxResult := range govSignedTxResultsCh {
	//	// Ignore any message that is NOT a param update
	//}
}
