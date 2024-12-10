package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

var _ client.SupplierQueryClient = (*supplierQuerier)(nil)

// supplierQuerier is a wrapper around the suppliertypes.QueryClient that enables the
// querying of on-chain supplier information through a single exposed method
// which returns an sharedtypes.Supplier struct
type supplierQuerier struct {
	client.ParamsQuerier[*suppliertypes.Params]

	clientConn      grpc.ClientConn
	supplierQuerier suppliertypes.QueryClient
}

// NewSupplierQuerier returns a new instance of a client.SupplierQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewSupplierQuerier(
	deps depinject.Config,
	paramsQuerierOpts ...ParamsQuerierOptionFn,
) (client.SupplierQueryClient, error) {
	paramsQuerierCfg := DefaultParamsQuerierConfig()
	for _, opt := range paramsQuerierOpts {
		opt(paramsQuerierCfg)
	}

	paramsQuerier, err := NewCachedParamsQuerier[*suppliertypes.Params, suppliertypes.SupplierQueryClient](
		deps, suppliertypes.NewSupplierQueryClient,
		WithModuleInfo[*suppliertypes.Params](suppliertypes.ModuleName, suppliertypes.ErrSupplierParamInvalid),
		WithParamsCacheOptions(paramsQuerierCfg.CacheOpts...),
	)
	if err != nil {
		return nil, err
	}

	sq := &supplierQuerier{
		ParamsQuerier: paramsQuerier,
	}

	if err = depinject.Inject(
		deps,
		&sq.clientConn,
	); err != nil {
		return nil, err
	}

	sq.supplierQuerier = suppliertypes.NewQueryClient(sq.clientConn)

	return sq, nil
}

// GetSupplier returns an suppliertypes.Supplier struct for a given address
func (sq *supplierQuerier) GetSupplier(
	ctx context.Context,
	operatorAddress string,
) (sharedtypes.Supplier, error) {
	req := &suppliertypes.QueryGetSupplierRequest{OperatorAddress: operatorAddress}
	res, err := sq.supplierQuerier.Supplier(ctx, req)
	if err != nil {
		return sharedtypes.Supplier{}, suppliertypes.ErrSupplierNotFound.Wrapf(
			"address: %s [%v]",
			operatorAddress, err,
		)
	}
	return res.Supplier, nil
}
