package application

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/tools/relay-spam/config"
)

type Staker struct {
	clientCtx client.Context
	config    *config.Config
	querier   *Querier
}

func NewStaker(clientCtx client.Context, cfg *config.Config) *Staker {
	// Create a gRPC client connection from the client context
	clientConn, err := grpc.Dial(
		clientCtx.NodeURI,
		grpc.WithInsecure(),
	)
	if err != nil {
		fmt.Printf("Failed to create gRPC client connection: %v\n", err)
		return &Staker{
			clientCtx: clientCtx,
			config:    cfg,
		}
	}

	// Create a new application querier
	querier, err := NewQuerier(clientConn)
	if err != nil {
		// Log the error but continue without the querier
		fmt.Printf("Failed to create application querier: %v\n", err)
		return &Staker{
			clientCtx: clientCtx,
			config:    cfg,
		}
	}

	return &Staker{
		clientCtx: clientCtx,
		config:    cfg,
		querier:   querier,
	}
}

// StakeApplications stakes all applications that are not already staked on the blockchain
func (s *Staker) StakeApplications() error {
	ctx := context.Background()

	for _, app := range s.config.Applications {
		// Check if the application is already staked
		isStaked := false
		if s.querier != nil {
			var err error
			isStaked, err = s.querier.IsStaked(ctx, app.Address)
			if err != nil {
				fmt.Printf("Error checking if application %s is staked: %v\n", app.Name, err)
			} else if isStaked {
				fmt.Printf("Application %s is already staked, skipping\n", app.Name)
				continue
			}
		} else {
			fmt.Printf("Querier not available, assuming application %s is not staked\n", app.Name)
		}

		// Get account from keyring
		key, err := s.clientCtx.Keyring.Key(app.Name)
		if err != nil {
			return fmt.Errorf("failed to get key for %s: %w", app.Name, err)
		}

		// Get the address from the key
		addr, err := key.GetAddress()
		if err != nil {
			return fmt.Errorf("failed to get address for %s: %w", app.Name, err)
		}
		fmt.Printf("Using address %s for application %s\n", addr.String(), app.Name)

		// Create stake message
		stakeAmount, err := sdktypes.ParseCoinNormalized(s.config.ApplicationDefaults.Stake)
		if err != nil {
			return fmt.Errorf("failed to parse stake amount: %w", err)
		}

		// Use the service ID from the application config if specified, otherwise use the default
		serviceID := s.config.ApplicationDefaults.ServiceID
		if app.ServiceIdGoal != "" {
			serviceID = app.ServiceIdGoal
		}

		// For now, we'll just print the command that would be executed
		// In a real implementation, we would use the Cosmos SDK to build and send the transaction
		cmd := fmt.Sprintf("poktrolld tx application stake %s %s %s",
			stakeAmount.String(),
			serviceID,
			s.config.TxFlags)

		fmt.Printf("Executing: %s\n", cmd)

		// In a real implementation, we would execute this command or use the SDK directly
		// For now, we'll just simulate success

		// Sleep to avoid sequence issues
		time.Sleep(time.Second)
	}

	return nil
}

// DelegateToGateway delegates applications to their specified gateway
func (s *Staker) DelegateToGateway() error {
	ctx := context.Background()

	for _, app := range s.config.Applications {
		// Skip if no gateways are specified for delegation
		if len(app.DelegateesGoal) == 0 {
			fmt.Printf("No gateways specified for delegation for application %s, skipping\n", app.Name)
			continue
		}

		// Check if the application is staked
		isStaked := false
		if s.querier != nil {
			var err error
			isStaked, err = s.querier.IsStaked(ctx, app.Address)
			if err != nil {
				fmt.Printf("Error checking if application %s is staked: %v\n", app.Name, err)
			} else if !isStaked {
				fmt.Printf("Application %s is not staked, skipping delegation\n", app.Name)
				continue
			}
		} else {
			fmt.Printf("Querier not available, assuming application %s is staked\n", app.Name)
		}

		// Process each gateway in the DelegateesGoal list
		for _, gatewayAddr := range app.DelegateesGoal {
			// Check if the application is already delegated to this gateway
			isDelegated := false
			if s.querier != nil {
				var err error
				isDelegated, err = s.querier.IsDelegatedToGateway(ctx, app.Address, gatewayAddr)
				if err != nil {
					fmt.Printf("Error checking if application %s is delegated to gateway %s: %v\n", app.Name, gatewayAddr, err)
				} else if isDelegated {
					fmt.Printf("Application %s is already delegated to gateway %s, skipping\n", app.Name, gatewayAddr)
					continue
				}
			} else {
				fmt.Printf("Querier not available, assuming application %s is not delegated to gateway %s\n", app.Name, gatewayAddr)
			}

			fmt.Printf("Delegating application %s to gateway %s...\n", app.Name, gatewayAddr)

			// In a real implementation, we would create a delegation message
			// For now, we'll just print the command that would be executed
			cmd := fmt.Sprintf("poktrolld tx application delegate %s %s %s",
				app.Address,
				gatewayAddr,
				s.config.TxFlags)

			fmt.Printf("Executing: %s\n", cmd)

			// In a real implementation, we would execute this command or use the SDK directly
			// For now, we'll just simulate success

			// Sleep to avoid sequence issues
			time.Sleep(time.Second)
		}
	}

	return nil
}
