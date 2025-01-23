package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// supplierQuerier is a wrapper around the suppliertypes.QueryClient that enables the
// querying of onchain supplier information through a single exposed method
// which returns an sharedtypes.Supplier struct
type supplierQuerier struct {
	clientConn      grpc.ClientConn
	supplierQuerier suppliertypes.QueryClient

	blockClient     client.BlockClient
	supplierCache   map[string]*sharedtypes.Supplier
	supplierCacheMu sync.Mutex
}

// NewSupplierQuerier returns a new instance of a client.SupplierQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewSupplierQuerier(ctx context.Context, deps depinject.Config) (client.SupplierQueryClient, error) {
	supq := &supplierQuerier{}

	if err := depinject.Inject(
		deps,
		&supq.blockClient,
		&supq.clientConn,
	); err != nil {
		return nil, err
	}

	supq.supplierQuerier = suppliertypes.NewQueryClient(supq.clientConn)

	channel.ForEach(
		ctx,
		supq.blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			supq.supplierCacheMu.Lock()
			defer supq.supplierCacheMu.Unlock()

			supq.supplierCache = make(map[string]*sharedtypes.Supplier)
		},
	)

	return supq, nil
}

// GetSupplier returns an suppliertypes.Supplier struct for a given address
func (supq *supplierQuerier) GetSupplier(
	ctx context.Context,
	operatorAddress string,
) (sharedtypes.Supplier, error) {
	supq.supplierCacheMu.Lock()
	defer supq.supplierCacheMu.Unlock()

	if supplier, ok := supq.supplierCache[operatorAddress]; ok {
		return *supplier, nil
	}

	req := &suppliertypes.QueryGetSupplierRequest{OperatorAddress: operatorAddress}
	res, err := supq.supplierQuerier.Supplier(ctx, req)
	if err != nil {
		return sharedtypes.Supplier{}, suppliertypes.ErrSupplierNotFound.Wrapf(
			"address: %s [%v]",
			operatorAddress, err,
		)
	}

	supq.supplierCache[operatorAddress] = &res.Supplier
	return res.Supplier, nil
}
