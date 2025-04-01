package application

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"cosmossdk.io/math"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pokt-network/poktroll/app/volatile"
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
	var clientConn *grpc.ClientConn
	var err error

	// Check if the endpoint uses port 443 (HTTPS)
	if strings.Contains(clientCtx.NodeURI, ":443") {
		// Use secure credentials for HTTPS endpoints
		clientConn, err = grpc.Dial(
			clientCtx.NodeURI,
			grpc.WithTransportCredentials(
				credentials.NewTLS(&tls.Config{
					InsecureSkipVerify: false,
				}),
			),
		)
	} else {
		// Use insecure credentials for non-HTTPS endpoints
		clientConn, err = grpc.Dial(
			clientCtx.NodeURI,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

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

// GetConfig returns the config for the staker
func (s *Staker) GetConfig() *config.Config {
	return s.config
}

// StakeApplication stakes a single application
func (s *Staker) StakeApplication(app config.Application) error {
	ctx := context.Background()

	// Parse the stake amount from the global application stake goal
	stakeAmount, err := sdk.ParseCoinNormalized(s.config.ApplicationStakeGoal)
	if err != nil {
		return fmt.Errorf("failed to parse stake amount: %w", err)
	}

	// Use the service ID from the application's serviceidgoal
	serviceID := app.ServiceIdGoal
	if serviceID == "" {
		return fmt.Errorf("serviceidgoal is required for application %s", app.Name)
	}

	// Check if the application is already staked
	isStaked := false
	if s.querier != nil {
		var err error
		isStaked, err = s.querier.IsStaked(ctx, app.Address)
		if err != nil {
			// If we get an error checking if the application is staked, log it but continue
			// This is likely because the application doesn't exist yet, which means it's not staked
			fmt.Printf("Warning: Error checking if application %s is staked: %v\n", app.Name, err)
			fmt.Printf("Assuming application %s is not staked, proceeding with staking\n", app.Name)
			isStaked = false
		} else if isStaked {
			// Check if the application is staked with the same amount
			isStakedWithAmount, err := s.querier.IsStakedWithAmount(ctx, app.Address, stakeAmount)
			if err != nil {
				fmt.Printf("Warning: Error checking if application %s is staked with amount %s: %v\n", app.Name, stakeAmount.String(), err)
			} else if isStakedWithAmount {
				// Check if the application is staked for the same service
				isStakedForService, err := s.querier.IsStakedForService(ctx, app.Address, serviceID)
				if err != nil {
					fmt.Printf("Warning: Error checking if application %s is staked for service %s: %v\n", app.Name, serviceID, err)
				} else if isStakedForService {
					fmt.Printf("Application %s is already staked with %s for service %s, skipping\n", app.Name, stakeAmount.String(), serviceID)
					return nil
				} else {
					fmt.Printf("Application %s is staked with %s but not for service %s, proceeding with staking\n", app.Name, stakeAmount.String(), serviceID)
				}
			} else {
				fmt.Printf("Application %s is staked but not with %s, proceeding with staking\n", app.Name, stakeAmount.String())
			}
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

	// Set fee amount based on gas limit and gas prices
	gasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)))
	fees := sdk.NewCoins()
	for _, gasPrice := range gasPrices {
		fee := gasPrice.Amount.MulInt(math.NewInt(int64(txBuilder.GetTx().GetGas()))).RoundInt()
		fees = fees.Add(sdk.NewCoin(gasPrice.Denom, fee))
	}
	txBuilder.SetFeeAmount(fees)

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

	return nil
}

// DelegateApplicationToGateway delegates a single application to its specified gateways
func (s *Staker) DelegateApplicationToGateway(app config.Application) error {
	ctx := context.Background()

	// Skip if no gateways are specified for delegation
	if len(app.DelegateesGoal) == 0 {
		fmt.Printf("No gateways specified for delegation for application %s, skipping\n", app.Name)
		return nil
	}

	// Check if the application is staked
	isStaked := false
	if s.querier != nil {
		var err error
		isStaked, err = s.querier.IsStaked(ctx, app.Address)
		if err != nil {
			// If we get an error checking if the application is staked, log it but continue
			// This is likely because the application doesn't exist yet
			fmt.Printf("Warning: Error checking if application %s is staked: %v\n", app.Name, err)
			fmt.Printf("Application %s may not be staked yet, skipping delegation\n", app.Name)
			return nil
		} else if !isStaked {
			fmt.Printf("Application %s is not staked, skipping delegation\n", app.Name)
			return nil
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

	// Delegate to each gateway
	for _, gatewayAddr := range app.DelegateesGoal {
		// Check if the application is already delegated to this gateway
		isDelegated := false
		if s.querier != nil {
			var err error
			isDelegated, err = s.querier.IsDelegatedToGateway(ctx, app.Address, gatewayAddr)
			if err != nil {
				// If we get an error checking if the application is delegated, log it but continue
				// This is likely because the application doesn't exist yet or there's a network issue
				fmt.Printf("Warning: Error checking if application %s is delegated to gateway %s: %v\n", app.Name, gatewayAddr, err)
				fmt.Printf("Assuming application %s is not delegated to gateway %s, proceeding with delegation\n", app.Name, gatewayAddr)
				isDelegated = false
			} else if isDelegated {
				fmt.Printf("Application %s is already delegated to gateway %s, skipping\n", app.Name, gatewayAddr)
				continue
			}
		} else {
			fmt.Printf("Querier not available, assuming application %s is not delegated to gateway %s\n", app.Name, gatewayAddr)
		}

		// Create the delegate application message
		msg := apptypes.NewMsgDelegateToGateway(
			addr.String(),
			gatewayAddr,
		)

		fmt.Printf("Delegating application %s to gateway %s...\n", app.Name, gatewayAddr)

		// Use the traditional approach to sign and broadcast the transaction
		txBuilder := s.clientCtx.TxConfig.NewTxBuilder()
		if err := txBuilder.SetMsgs(msg); err != nil {
			return fmt.Errorf("failed to set messages: %w", err)
		}

		// Set gas limit - using a high value to ensure it goes through
		txBuilder.SetGasLimit(1000000)

		// Set fee amount based on gas limit and gas prices
		gasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)))
		fees := sdk.NewCoins()
		for _, gasPrice := range gasPrices {
			fee := gasPrice.Amount.MulInt(math.NewInt(int64(txBuilder.GetTx().GetGas()))).RoundInt()
			fees = fees.Add(sdk.NewCoin(gasPrice.Denom, fee))
		}
		txBuilder.SetFeeAmount(fees)

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

	return nil
}

// StakeApplications stakes all applications that are not already staked on the blockchain
func (s *Staker) StakeApplications() error {
	ctx := context.Background()

	for _, app := range s.config.Applications {
		// Parse the stake amount from the global application stake goal
		stakeAmount, err := sdk.ParseCoinNormalized(s.config.ApplicationStakeGoal)
		if err != nil {
			return fmt.Errorf("failed to parse stake amount: %w", err)
		}

		// Use the service ID from the application's serviceidgoal
		serviceID := app.ServiceIdGoal
		if serviceID == "" {
			return fmt.Errorf("serviceidgoal is required for application %s", app.Name)
		}

		// Check if the application is already staked
		isStaked := false
		if s.querier != nil {
			var err error
			isStaked, err = s.querier.IsStaked(ctx, app.Address)
			if err != nil {
				// If we get an error checking if the application is staked, log it but continue
				// This is likely because the application doesn't exist yet, which means it's not staked
				fmt.Printf("Warning: Error checking if application %s is staked: %v\n", app.Name, err)
				fmt.Printf("Assuming application %s is not staked, proceeding with staking\n", app.Name)
				isStaked = false
			} else if isStaked {
				// Check if the application is staked with the same amount
				isStakedWithAmount, err := s.querier.IsStakedWithAmount(ctx, app.Address, stakeAmount)
				if err != nil {
					fmt.Printf("Warning: Error checking if application %s is staked with amount %s: %v\n", app.Name, stakeAmount.String(), err)
				} else if isStakedWithAmount {
					// Check if the application is staked for the same service
					isStakedForService, err := s.querier.IsStakedForService(ctx, app.Address, serviceID)
					if err != nil {
						fmt.Printf("Warning: Error checking if application %s is staked for service %s: %v\n", app.Name, serviceID, err)
					} else if isStakedForService {
						fmt.Printf("Application %s is already staked with %s for service %s, skipping\n", app.Name, stakeAmount.String(), serviceID)
						continue
					} else {
						fmt.Printf("Application %s is staked with %s but not for service %s, proceeding with staking\n", app.Name, stakeAmount.String(), serviceID)
					}
				} else {
					fmt.Printf("Application %s is staked but not with %s, proceeding with staking\n", app.Name, stakeAmount.String())
				}
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

		// Set fee amount based on gas limit and gas prices
		gasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)))
		fees := sdk.NewCoins()
		for _, gasPrice := range gasPrices {
			fee := gasPrice.Amount.MulInt(math.NewInt(int64(txBuilder.GetTx().GetGas()))).RoundInt()
			fees = fees.Add(sdk.NewCoin(gasPrice.Denom, fee))
		}
		txBuilder.SetFeeAmount(fees)

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
				// If we get an error checking if the application is staked, log it but continue
				// This is likely because the application doesn't exist yet
				fmt.Printf("Warning: Error checking if application %s is staked: %v\n", app.Name, err)
				fmt.Printf("Application %s may not be staked yet, skipping delegation\n", app.Name)
				continue
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
					// If we get an error checking if the application is delegated, log it but continue
					// This is likely because the application doesn't exist yet or there's a network issue
					fmt.Printf("Warning: Error checking if application %s is delegated to gateway %s: %v\n", app.Name, gatewayAddr, err)
					fmt.Printf("Assuming application %s is not delegated to gateway %s, proceeding with delegation\n", app.Name, gatewayAddr)
					isDelegated = false
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

			// Set fee amount based on gas limit and gas prices
			gasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(volatile.DenomuPOKT, math.LegacyNewDecWithPrec(1, 2)))
			fees := sdk.NewCoins()
			for _, gasPrice := range gasPrices {
				fee := gasPrice.Amount.MulInt(math.NewInt(int64(txBuilder.GetTx().GetGas()))).RoundInt()
				fees = fees.Add(sdk.NewCoin(gasPrice.Denom, fee))
			}
			txBuilder.SetFeeAmount(fees)

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

// Querier returns the querier instance for the staker
func (s *Staker) Querier() *Querier {
	return s.querier
}
