package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// supplierQuerier is a wrapper around the suppliertypes.QueryClient that enables the
// querying of onchain supplier information through a single exposed method
// which returns an sharedtypes.Supplier struct
type supplierQuerier struct {
	clientConn      grpc.ClientConn
	supplierQuerier suppliertypes.QueryClient
}

// NewSupplierQuerier returns a new instance of a client.SupplierQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewSupplierQuerier(deps depinject.Config) (client.SupplierQueryClient, error) {
	supq := &supplierQuerier{}

	if err := depinject.Inject(
		deps,
		&supq.clientConn,
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
	req := &suppliertypes.QueryGetSupplierRequest{OperatorAddress: operatorAddress}
	res, err := supq.supplierQuerier.Supplier(ctx, req)
	if err != nil {
		return sharedtypes.Supplier{}, err
	}
	return res.Supplier, nil
}

// GetParams returns the supplier module parameters.
func (supq *supplierQuerier) GetParams(ctx context.Context) (*suppliertypes.Params, error) {
	req := suppliertypes.QueryParamsRequest{}
	res, err := supq.supplierQuerier.Params(ctx, &req)
	if err != nil {
		return nil, err
	}
	return &res.Params, nil
}
