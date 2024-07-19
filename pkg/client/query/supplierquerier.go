package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/proto/types/supplier"
)

// supplierQuerier is a wrapper around the suppliertypes.QueryClient that enables the
// querying of on-chain supplier information through a single exposed method
// which returns an sharedtypes.Supplier struct
type supplierQuerier struct {
	clientConn      grpc.ClientConn
	supplierQuerier supplier.QueryClient
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

	supq.supplierQuerier = supplier.NewQueryClient(supq.clientConn)

	return supq, nil
}

// GetSupplier returns an suppliertypes.Supplier struct for a given address
func (supq *supplierQuerier) GetSupplier(
	ctx context.Context,
	address string,
) (shared.Supplier, error) {
	req := &supplier.QueryGetSupplierRequest{Address: address}
	res, err := supq.supplierQuerier.Supplier(ctx, req)
	if err != nil {
		return shared.Supplier{}, supplier.ErrSupplierNotFound.Wrapf(
			"address: %s [%v]",
			address, err,
		)
	}
	return res.Supplier, nil
}
