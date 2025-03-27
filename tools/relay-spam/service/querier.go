package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Querier provides methods to query service data from the blockchain
type Querier struct {
	serviceQueryClient servicetypes.QueryClient
}

// NewQuerier creates a new service querier
func NewQuerier(clientConn *grpc.ClientConn) (*Querier, error) {
	if clientConn == nil {
		return nil, fmt.Errorf("client connection is nil")
	}

	// Create service query client directly from the gRPC connection
	serviceQueryClient := servicetypes.NewQueryClient(clientConn)

	return &Querier{
		serviceQueryClient: serviceQueryClient,
	}, nil
}

// ServiceExists checks if a service with the given ID exists
func (q *Querier) ServiceExists(ctx context.Context, serviceID string) (bool, error) {
	// Create a QueryGetServiceRequest with the service ID
	req := &servicetypes.QueryGetServiceRequest{
		Id: serviceID,
	}

	// Call the Service method
	_, err := q.serviceQueryClient.Service(ctx, req)
	if err != nil {
		// Check for "service not found" error in different ways
		errStr := err.Error()
		if errStr == servicetypes.ErrServiceNotFound.Error() ||
			errStr == "service not found" ||
			errStr == "rpc error: code = NotFound desc = service not found" ||
			errStr == fmt.Sprintf("service id: %s [rpc error: code = NotFound desc = service not found]", serviceID) {
			// If the service is not found, it doesn't exist
			return false, nil
		}
		return false, fmt.Errorf("failed to get service: %w", err)
	}

	// If we got a service, it exists
	return true, nil
}

// GetService returns the service with the given ID
func (q *Querier) GetService(ctx context.Context, serviceID string) (sharedtypes.Service, error) {
	// Create a QueryGetServiceRequest with the service ID
	req := &servicetypes.QueryGetServiceRequest{
		Id: serviceID,
	}

	// Call the Service method
	resp, err := q.serviceQueryClient.Service(ctx, req)
	if err != nil {
		return sharedtypes.Service{}, fmt.Errorf("failed to get service: %w", err)
	}

	return resp.Service, nil
}
