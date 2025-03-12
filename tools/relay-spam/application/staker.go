package application

import (
	"context"
	"fmt"
	"time"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/tools/relay-spam/config"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

type Staker struct {
	clientCtx cosmosclient.Context
	config    *config.Config
	querier   *Querier
}

func NewStaker(clientCtx cosmosclient.Context, cfg *config.Config) *Staker {
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
	// We need to pass the clientConn as an interface, not a pointer
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
		stakeAmount, err := sdk.ParseCoinNormalized(s.config.ApplicationDefaults.Stake)
		if err != nil {
			return fmt.Errorf("failed to parse stake amount: %w", err)
		}

		// Use the service ID from the application config if specified, otherwise use the default
		serviceID := s.config.ApplicationDefaults.ServiceID
		if app.ServiceIdGoal != "" {
			serviceID = app.ServiceIdGoal
		}

		// Create service configs
		services := []*sharedtypes.ApplicationServiceConfig{
			{
				ServiceId: serviceID,
			},
		}

		// Create the stake application message
		msg := apptypes.NewMsgStakeApplication(
			addr.String(),
			stakeAmount,
			services,
		)

		fmt.Printf("Staking application %s with %s for service %s...\n", app.Name, stakeAmount.String(), serviceID)

		// Use the traditional approach to sign and broadcast the transaction
		txBuilder := s.clientCtx.TxConfig.NewTxBuilder()
		if err := txBuilder.SetMsgs(msg); err != nil {
			return fmt.Errorf("failed to set messages: %w", err)
		}

		// Set gas limit - using a high value to ensure it goes through
		txBuilder.SetGasLimit(1000000)

		// Get account number and sequence
		accNum, accSeq, err := s.clientCtx.AccountRetriever.GetAccountNumberSequence(s.clientCtx, addr)
		if err != nil {
			return fmt.Errorf("failed to get account number and sequence: %w", err)
		}

		// Create a transaction factory
		txFactory := cosmostx.Factory{}.
			WithChainID(s.clientCtx.ChainID).
			WithKeybase(s.clientCtx.Keyring).
			WithTxConfig(s.clientCtx.TxConfig).
			WithAccountRetriever(s.clientCtx.AccountRetriever).
			WithAccountNumber(accNum).
			WithSequence(accSeq)

		// Sign the transaction
		err = cosmostx.Sign(ctx, txFactory, app.Name, txBuilder, true)
		if err != nil {
			return fmt.Errorf("failed to sign transaction: %w", err)
		}

		// Encode the transaction
		txBytes, err := s.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
		if err != nil {
			return fmt.Errorf("failed to encode transaction: %w", err)
		}

		// Broadcast the transaction
		res, err := s.clientCtx.BroadcastTxSync(txBytes)
		if err != nil {
			return fmt.Errorf("failed to broadcast transaction: %w", err)
		}

		// Check for errors in the response
		if res.Code != 0 {
			return fmt.Errorf("transaction failed: %s", res.RawLog)
		}

		fmt.Printf("Successfully staked application %s. Transaction hash: %s\n", app.Name, res.TxHash)

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

			// Create the delegate to gateway message
			msg := apptypes.NewMsgDelegateToGateway(
				addr.String(),
				gatewayAddr,
			)

			// Use the traditional approach to sign and broadcast the transaction
			txBuilder := s.clientCtx.TxConfig.NewTxBuilder()
			if err := txBuilder.SetMsgs(msg); err != nil {
				return fmt.Errorf("failed to set messages: %w", err)
			}

			// Set gas limit - using a high value to ensure it goes through
			txBuilder.SetGasLimit(1000000)

			// Get account number and sequence
			accNum, accSeq, err := s.clientCtx.AccountRetriever.GetAccountNumberSequence(s.clientCtx, addr)
			if err != nil {
				return fmt.Errorf("failed to get account number and sequence: %w", err)
			}

			// Create a transaction factory
			txFactory := cosmostx.Factory{}.
				WithChainID(s.clientCtx.ChainID).
				WithKeybase(s.clientCtx.Keyring).
				WithTxConfig(s.clientCtx.TxConfig).
				WithAccountRetriever(s.clientCtx.AccountRetriever).
				WithAccountNumber(accNum).
				WithSequence(accSeq)

			// Sign the transaction
			err = cosmostx.Sign(ctx, txFactory, app.Name, txBuilder, true)
			if err != nil {
				return fmt.Errorf("failed to sign transaction: %w", err)
			}

			// Encode the transaction
			txBytes, err := s.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
			if err != nil {
				return fmt.Errorf("failed to encode transaction: %w", err)
			}

			// Broadcast the transaction
			res, err := s.clientCtx.BroadcastTxSync(txBytes)
			if err != nil {
				return fmt.Errorf("failed to broadcast transaction: %w", err)
			}

			// Check for errors in the response
			if res.Code != 0 {
				return fmt.Errorf("transaction failed: %s", res.RawLog)
			}

			fmt.Printf("Successfully delegated application %s to gateway %s. Transaction hash: %s\n", app.Name, gatewayAddr, res.TxHash)

			// Sleep to avoid sequence issues
			time.Sleep(time.Second)
		}
	}

	return nil
}
