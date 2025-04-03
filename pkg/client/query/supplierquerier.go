package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/cache"
	"github.com/pokt-network/poktroll/pkg/client"
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
}

// NewSupplierQuerier returns a new instance of a client.SupplierQueryClient by
// injecting the dependencies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
// - polylog.Logger
// - cache.KeyValueCache[sharedtypes.Supplier]
func NewSupplierQuerier(deps depinject.Config) (client.SupplierQueryClient, error) {
	supq := &supplierQuerier{}

	if err := depinject.Inject(
		deps,
		&supq.clientConn,
		&supq.logger,
		&supq.suppliersCache,
	); err != nil {
		return nil, err
	}

	supq.supplierQuerier = suppliertypes.NewQueryClient(supq.clientConn)

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
		logger.Debug().Msgf("cache hit for operator address key: %s", operatorAddress)
		return supplier, nil
	}

	logger.Debug().Msgf("cache miss for operator address key: %s", operatorAddress)

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
	req := suppliertypes.QueryParamsRequest{}
	res, err := retry.Call(ctx, func() (*suppliertypes.QueryParamsResponse, error) {
		return supq.supplierQuerier.Params(ctx, &req)
	}, retry.GetStrategy(ctx))
	if err != nil {
		return nil, err
	}
	return &res.Params, nil
}
