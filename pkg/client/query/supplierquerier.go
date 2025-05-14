package query

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"
	proto "github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/retry"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// supplierQuerier is a wrapper around the suppliertypes.QueryClient that enables the
// querying of onchain supplier information through a single exposed method
// which returns an sharedtypes.Supplier struct
type supplierQuerier struct {
	clientConn      grpc.ClientConn
	supplierQuerier suppliertypes.QueryClient
	logger          polylog.Logger

	// suppliersCache caches supplierQueryClient.Supplier requests
	suppliersCache cache.KeyValueCache[sharedtypes.Supplier]
	// suppliersMutex to protect cache access patterns for suppliers
	suppliersMutex sync.Mutex

	// eventsParamsActivationClient is used to subscribe to supplier module parameters updates
	eventsParamsActivationClient client.EventsParamsActivationClient
	// paramsCache caches supplier module parameters
	paramsCache client.ParamsCache[suppliertypes.Params]
}

// NewSupplierQuerier returns a new instance of a client.SupplierQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
// - polylog.Logger
// - client.EventsParamsActivationClient
// - cache.KeyValueCache[sharedtypes.Supplier]
// - client.ParamsCache[suppliertypes.Params]
func NewSupplierQuerier(
	ctx context.Context,
	deps depinject.Config,
) (client.SupplierQueryClient, error) {
	supq := &supplierQuerier{}

	if err := depinject.Inject(
		deps,
		&supq.clientConn,
		&supq.logger,
		&supq.eventsParamsActivationClient,
		&supq.suppliersCache,
		&supq.paramsCache,
	); err != nil {
		return nil, err
	}

	supq.supplierQuerier = suppliertypes.NewQueryClient(supq.clientConn)

	// Initialize the supplier module cache with all existing parameters updates:
	// - Parameters are cached as historic data, eliminating the need to invalidate the cache.
	// - The UpdateParamsCache method ensures the querier starts with the current parameters history cached.
	// - Future updates are automatically cached by subscribing to the eventsParamsActivationClient observable.
	err := querycache.UpdateParamsCache(
		ctx,
		&suppliertypes.QueryParamsUpdatesRequest{},
		toSupplierParamsUpdate,
		supq.supplierQuerier,
		supq.eventsParamsActivationClient,
		supq.paramsCache,
	)
	if err != nil {
		return nil, err
	}

	return supq, nil
}

// GetSupplier returns an suppliertypes.Supplier struct for a given address
func (supq *supplierQuerier) GetSupplier(
	ctx context.Context,
	operatorAddress string,
) (sharedtypes.Supplier, error) {
	logger := supq.logger.With("query_client", "supplier", "method", "GetSupplier")

	// Check if the supplier is present in the cache.
	if supplier, found := supq.suppliersCache.Get(operatorAddress); found {
		logger.Debug().Msgf("cache HIT for operator address key: %s", operatorAddress)
		return supplier, nil
	}

	// Use mutex to prevent multiple concurrent cache updates
	supq.suppliersMutex.Lock()
	defer supq.suppliersMutex.Unlock()

	// Double-check cache after acquiring lock (follows standard double-checked locking pattern)
	if supplier, found := supq.suppliersCache.Get(operatorAddress); found {
		logger.Debug().Msgf("cache HIT for operator address key after lock: %s", operatorAddress)
		return supplier, nil
	}

	logger.Debug().Msgf("cache MISS for operator address key: %s", operatorAddress)

	req := &suppliertypes.QueryGetSupplierRequest{OperatorAddress: operatorAddress}
	res, err := retry.Call(ctx, func() (*suppliertypes.QueryGetSupplierResponse, error) {
		return supq.supplierQuerier.Supplier(ctx, req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return sharedtypes.Supplier{}, err
	}

	// Cache the supplier for future use.
	supq.suppliersCache.Set(operatorAddress, res.Supplier)
	return res.Supplier, nil
}

// GetParams returns the supplier module parameters.
func (supq *supplierQuerier) GetParams(ctx context.Context) (*suppliertypes.Params, error) {
	logger := supq.logger.With("query_client", "supplier", "method", "GetParams")

	// Attempt to retrieve the latest parameters from the cache.
	params, found := supq.paramsCache.GetLatest()
	if !found {
		logger.Debug().Msg("cache MISS for supplier params")
		return nil, fmt.Errorf("expecting supplier params to be found in cache")
	}

	logger.Debug().Msg("cache HIT for supplier params")

	return &params, nil
}

func toSupplierParamsUpdate(protoMessage proto.Message) (*suppliertypes.ParamsUpdate, bool) {
	if event, ok := protoMessage.(*suppliertypes.EventParamsActivated); ok {
		return &event.ParamsUpdate, true
	}

	return nil, false
}
