package query

import (
	"context"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// supplierQuerier is a wrapper around the suppliertypes.QueryClient that enables the
// querying of on-chain supplier information through a single exposed method
// which returns an sharedtypes.Supplier struct
type supplierQuerier struct {
	clientCtx       types.Context
	supplierQuerier suppliertypes.QueryClient
}

// NewSupplierQuerier returns a new instance of a client.SupplierQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx
func NewSupplierQuerier(deps depinject.Config) (client.SupplierQueryClient, error) {
	supq := &supplierQuerier{}

	if err := depinject.Inject(
		deps,
		&supq.clientCtx,
	); err != nil {
		return nil, err
	}

	supq.supplierQuerier = suppliertypes.NewQueryClient(cosmosclient.Context(supq.clientCtx))

	return supq, nil
}

// GetSupplier returns an suppliertypes.Supplier struct for a given address
func (supq *supplierQuerier) GetSupplier(
	ctx context.Context,
	address string,
) (sharedtypes.Supplier, error) {
	req := &suppliertypes.QueryGetSupplierRequest{Address: address}
	res, err := supq.supplierQuerier.Supplier(ctx, req)
	if err != nil {
		return sharedtypes.Supplier{}, suppliertypes.ErrSupplierNotFound.Wrapf(
			"address: %s [%v]",
			address, err,
		)
	}
	return res.Supplier, nil
}
