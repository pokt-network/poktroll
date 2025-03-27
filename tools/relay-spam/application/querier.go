package application

import (
	"context"
	"fmt"

	"cosmossdk.io/depinject"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/pkg/cache/memory"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// Querier provides methods to query application data from the blockchain
type Querier struct {
	appQueryClient client.ApplicationQueryClient
}

// NewQuerier creates a new application querier
func NewQuerier(clientConn *grpc.ClientConn) (*Querier, error) {
	// Create application cache
	appCache, err := memory.NewKeyValueCache[apptypes.Application]()
	if err != nil {
		return nil, fmt.Errorf("failed to create application cache: %w", err)
	}

	// Create application params cache
	appParamsCache, err := cache.NewParamsCache[apptypes.Params]()
	if err != nil {
		return nil, fmt.Errorf("failed to create application params cache: %w", err)
	}

	// Create logger
	logger := polylog.DefaultContextLogger

	// Create dependencies for application querier
	deps := depinject.Supply(clientConn, appCache, appParamsCache, logger)

	// Create application query client
	appQueryClient, err := query.NewApplicationQuerier(deps)
	if err != nil {
		return nil, fmt.Errorf("failed to create application query client: %w", err)
	}

	return &Querier{
		appQueryClient: appQueryClient,
	}, nil
}

// IsStaked checks if the application is staked
func (q *Querier) IsStaked(ctx context.Context, appAddress string) (bool, error) {
	app, err := q.appQueryClient.GetApplication(ctx, appAddress)
	if err != nil {
		// Check for "application not found" error in different ways
		errStr := err.Error()
		if errStr == apptypes.ErrAppNotFound.Error() ||
			errStr == "application not found" ||
			errStr == "rpc error: code = NotFound desc = application not found" ||
			errStr == "app address: "+appAddress+" [rpc error: code = NotFound desc = application not found]" {
			// If the application is not found, it's not staked
			return false, nil
		}
		return false, fmt.Errorf("failed to get application: %w", err)
	}

	// Check if the application has a stake
	return app.Stake != nil && app.Stake.Amount.IsPositive(), nil
}

// GetStake returns the current stake amount of the application
func (q *Querier) GetStake(ctx context.Context, appAddress string) (*sdk.Coin, error) {
	app, err := q.appQueryClient.GetApplication(ctx, appAddress)
	if err != nil {
		// Check for "application not found" error in different ways
		errStr := err.Error()
		if errStr == apptypes.ErrAppNotFound.Error() ||
			errStr == "application not found" ||
			errStr == "rpc error: code = NotFound desc = application not found" ||
			errStr == "app address: "+appAddress+" [rpc error: code = NotFound desc = application not found]" {
			// If the application is not found, return nil stake
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	return app.Stake, nil
}

// IsDelegated checks if the application is delegated to any gateway
func (q *Querier) IsDelegated(ctx context.Context, appAddress string) (bool, error) {
	delegatees, err := q.GetDelegatees(ctx, appAddress)
	if err != nil {
		return false, err
	}

	return len(delegatees) > 0, nil
}

// GetDelegatees returns the list of gateways that the application is delegated to
func (q *Querier) GetDelegatees(ctx context.Context, appAddress string) ([]string, error) {
	app, err := q.appQueryClient.GetApplication(ctx, appAddress)
	if err != nil {
		// Check for "application not found" error in different ways
		errStr := err.Error()
		if errStr == apptypes.ErrAppNotFound.Error() ||
			errStr == "application not found" ||
			errStr == "rpc error: code = NotFound desc = application not found" ||
			errStr == "app address: "+appAddress+" [rpc error: code = NotFound desc = application not found]" {
			// If the application is not found, return empty list
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	return app.DelegateeGatewayAddresses, nil
}

// IsDelegatedToGateway checks if the application is delegated to a specific gateway
func (q *Querier) IsDelegatedToGateway(ctx context.Context, appAddress, gatewayAddress string) (bool, error) {
	delegatees, err := q.GetDelegatees(ctx, appAddress)
	if err != nil {
		return false, err
	}

	for _, delegatee := range delegatees {
		if delegatee == gatewayAddress {
			return true, nil
		}
	}

	return false, nil
}

// GetServiceIDs returns the list of service IDs that the application is staked for
func (q *Querier) GetServiceIDs(ctx context.Context, appAddress string) ([]string, error) {
	app, err := q.appQueryClient.GetApplication(ctx, appAddress)
	if err != nil {
		// Check for "application not found" error in different ways
		errStr := err.Error()
		if errStr == apptypes.ErrAppNotFound.Error() ||
			errStr == "application not found" ||
			errStr == "rpc error: code = NotFound desc = application not found" ||
			errStr == "app address: "+appAddress+" [rpc error: code = NotFound desc = application not found]" {
			// If the application is not found, return empty list
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	serviceIDs := make([]string, 0, len(app.ServiceConfigs))
	for _, service := range app.ServiceConfigs {
		serviceIDs = append(serviceIDs, service.ServiceId)
	}

	return serviceIDs, nil
}

// IsStakedForService checks if the application is staked for a specific service
func (q *Querier) IsStakedForService(ctx context.Context, appAddress, serviceID string) (bool, error) {
	serviceIDs, err := q.GetServiceIDs(ctx, appAddress)
	if err != nil {
		return false, err
	}

	for _, id := range serviceIDs {
		if id == serviceID {
			return true, nil
		}
	}

	return false, nil
}

// IsStakedWithAmount checks if the application is staked with a specific amount
func (q *Querier) IsStakedWithAmount(ctx context.Context, appAddress string, amount sdk.Coin) (bool, error) {
	stake, err := q.GetStake(ctx, appAddress)
	if err != nil {
		return false, err
	}

	// If the stake is nil, the application is not staked
	if stake == nil {
		return false, nil
	}

	// Check if the stake amount is the same
	return stake.Denom == amount.Denom && stake.Amount.Equal(amount.Amount), nil
}
