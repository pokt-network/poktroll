package supplier

import (
	"context"
	"fmt"

	"cosmossdk.io/depinject"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// Querier provides methods to query supplier data from the blockchain
type Querier struct {
	supplierQueryClient client.SupplierQueryClient
}

// NewQuerier creates a new supplier querier
func NewQuerier(clientConn *grpc.ClientConn) (*Querier, error) {
	// Create supplier cache
	supplierCache, err := memory.NewKeyValueCache[sharedtypes.Supplier]()
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier cache: %w", err)
	}

	// Create supplier params cache
	supplierParamsCache, err := cache.NewParamsCache[suppliertypes.Params]()
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier params cache: %w", err)
	}

	// Create logger
	logger := polylog.DefaultContextLogger

	// Create dependencies for supplier querier
	deps := depinject.Supply(clientConn, supplierCache, supplierParamsCache, logger)

	// Create supplier query client
	supplierQueryClient, err := query.NewSupplierQuerier(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create supplier query client: %w", err)
	}

	return &Querier{
		supplierQueryClient: supplierQueryClient,
	}, nil
}

// SupplierExists checks if a supplier with the given address exists
func (q *Querier) SupplierExists(ctx context.Context, supplierAddress string) (bool, error) {
	_, err := q.supplierQueryClient.GetSupplier(ctx, supplierAddress)
	if err != nil {
		// Check for "supplier not found" error in different ways
		errStr := err.Error()
		if errStr == suppliertypes.ErrSupplierNotFound.Error() ||
			errStr == "supplier not found" ||
			errStr == "rpc error: code = NotFound desc = supplier not found" ||
			errStr == "supplier address: "+supplierAddress+" [rpc error: code = NotFound desc = supplier not found]" {
			// If the supplier is not found, it doesn't exist
			return false, nil
		}
		return false, fmt.Errorf("failed to get supplier: %w", err)
	}

	// If we got a supplier, it exists
	return true, nil
}

// GetSupplier returns the supplier with the given address
func (q *Querier) GetSupplier(ctx context.Context, supplierAddress string) (sharedtypes.Supplier, error) {
	return q.supplierQueryClient.GetSupplier(ctx, supplierAddress)
}
