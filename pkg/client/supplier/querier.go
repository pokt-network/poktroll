package supplier

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
	suppliertypes "github.com/pokt-network/poktroll/proto/types/supplier"
)

var _ client.SupplierQueryClient = (*supplierQueryClient)(nil)

// supplierQueryClient is a wrapper around the suppliertypes.QueryClient that enables the
// querying of on-chain supplier information through a single exposed method
// which returns an sharedtypes.Supplier struct
type supplierQueryClient struct {
	clientConn      grpc.ClientConn
	supplierQuerier suppliertypes.QueryClient
}

// NewSupplierQueryClient returns a new instance of a client.SupplierQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewSupplierQueryClient(deps depinject.Config) (client.SupplierQueryClient, error) {
	supq := &supplierQueryClient{}

	if err := depinject.Inject(
		deps,
		&supq.clientConn,
	); err != nil {
		return nil, err
	}

	supq.supplierQuerier = suppliertypes.NewQueryClient(supq.clientConn)

	return supq, nil
}

// GetSupplier returns the supplier for a given address
func (supq *supplierQueryClient) GetSupplier(
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
