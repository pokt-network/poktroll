package supplier

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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
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

	// Create a new supplier querier
	querier, err := NewQuerier(clientConn)
	if err != nil {
		// Log the error but continue without the querier
		fmt.Printf("Failed to create supplier querier: %v\n", err)
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

// StakeSupplier stakes a single supplier
func (s *Staker) StakeSupplier(supplier config.Supplier) error {
	ctx := context.Background()

	// Parse the stake amount from the global supplier stake goal
	stakeAmount, err := sdk.ParseCoinNormalized(s.config.SupplierStakeGoal)
	if err != nil {
		return fmt.Errorf("failed to parse stake amount: %w", err)
	}

	// Check if the supplier is already staked
	supplierExists := false
	if s.querier != nil {
		var err error
		supplierExists, err = s.querier.SupplierExists(ctx, supplier.Address)
		if err != nil {
			// If we get an error checking if the supplier exists, log it but continue
			fmt.Printf("Warning: Error checking if supplier %s exists: %v\n", supplier.Name, err)
			fmt.Printf("Assuming supplier %s does not exist, proceeding with staking\n", supplier.Name)
			supplierExists = false
		} else if supplierExists {
			fmt.Printf("Supplier %s already exists, skipping\n", supplier.Name)
			return nil
		}
	} else {
		fmt.Printf("Querier not available, assuming supplier %s is not staked\n", supplier.Name)
	}

	// Get account from keyring
	key, err := s.clientCtx.Keyring.Key(supplier.Name)
	if err != nil {
		return fmt.Errorf("failed to get key for %s: %w", supplier.Name, err)
	}

	// Get the address from the key
	addr, err := key.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get address for %s: %w", supplier.Name, err)
	}
	fmt.Printf("Using address %s for supplier %s\n", addr.String(), supplier.Name)

	// Create service configs from the stake config
	var services []*sharedtypes.SupplierServiceConfig
	for _, svcConfig := range supplier.StakeConfig.Services {
		// Convert endpoints from YAMLServiceEndpoint to SupplierEndpoint
		var endpoints []*sharedtypes.SupplierEndpoint
		for _, endpoint := range svcConfig.Endpoints {
			// Create a new SupplierEndpoint with the correct field names
			supplierEndpoint := &sharedtypes.SupplierEndpoint{
				Url:     endpoint.PubliclyExposedUrl,
				RpcType: sharedtypes.RPCType_JSON_RPC, // Default to JSON_RPC, adjust as needed
			}

			// Add configs if available
			if endpoint.Config != nil && len(endpoint.Config) > 0 {
				var configs []*sharedtypes.ConfigOption
				for key, value := range endpoint.Config {
					configs = append(configs, &sharedtypes.ConfigOption{
						Key:   sharedtypes.ConfigOptions_TIMEOUT, // Use TIMEOUT as it's the only available option
						Value: fmt.Sprintf("%s=%s", key, value),
					})
				}
				supplierEndpoint.Configs = configs
			}

			endpoints = append(endpoints, supplierEndpoint)
		}

		services = append(services, &sharedtypes.SupplierServiceConfig{
			ServiceId: svcConfig.ServiceId,
			Endpoints: endpoints,
		})
	}

	// Create the stake supplier message
	msg := suppliertypes.NewMsgStakeSupplier(
		supplier.OwnerAddress,
		addr.String(),
		supplier.Name,
		stakeAmount,
		services,
	)

	fmt.Printf("Staking supplier %s with %s...\n", supplier.Name, stakeAmount.String())

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
	err = cosmostx.Sign(ctx, txFactory, supplier.Name, txBuilder, true)
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

	fmt.Printf("Successfully staked supplier %s. Transaction hash: %s\n", supplier.Name, res.TxHash)

	// Sleep to avoid sequence issues
	time.Sleep(time.Second)

	return nil
}

// StakeSuppliers stakes all suppliers that are not already staked on the blockchain
func (s *Staker) StakeSuppliers() error {
	ctx := context.Background()

	for _, supplier := range s.config.Suppliers {
		// Parse the stake amount from the global supplier stake goal
		stakeAmount, err := sdk.ParseCoinNormalized(s.config.SupplierStakeGoal)
		if err != nil {
			return fmt.Errorf("failed to parse stake amount: %w", err)
		}

		// Check if the supplier is already staked
		supplierExists := false
		if s.querier != nil {
			var err error
			supplierExists, err = s.querier.SupplierExists(ctx, supplier.Address)
			if err != nil {
				// If we get an error checking if the supplier exists, log it but continue
				fmt.Printf("Warning: Error checking if supplier %s exists: %v\n", supplier.Name, err)
				fmt.Printf("Assuming supplier %s does not exist, proceeding with staking\n", supplier.Name)
				supplierExists = false
			} else if supplierExists {
				fmt.Printf("Supplier %s already exists, skipping\n", supplier.Name)
				continue
			}
		} else {
			fmt.Printf("Querier not available, assuming supplier %s is not staked\n", supplier.Name)
		}

		// Get account from keyring
		key, err := s.clientCtx.Keyring.Key(supplier.Name)
		if err != nil {
			return fmt.Errorf("failed to get key for %s: %w", supplier.Name, err)
		}

		// Get the address from the key
		addr, err := key.GetAddress()
		if err != nil {
			return fmt.Errorf("failed to get address for %s: %w", supplier.Name, err)
		}
		fmt.Printf("Using address %s for supplier %s\n", addr.String(), supplier.Name)

		// Create service configs from the stake config
		var services []*sharedtypes.SupplierServiceConfig
		for _, svcConfig := range supplier.StakeConfig.Services {
			// Convert endpoints from YAMLServiceEndpoint to SupplierEndpoint
			var endpoints []*sharedtypes.SupplierEndpoint
			for _, endpoint := range svcConfig.Endpoints {
				// Create a new SupplierEndpoint with the correct field names
				supplierEndpoint := &sharedtypes.SupplierEndpoint{
					Url:     endpoint.PubliclyExposedUrl,
					RpcType: sharedtypes.RPCType_JSON_RPC, // Default to JSON_RPC, adjust as needed
				}

				// Add configs if available
				if endpoint.Config != nil && len(endpoint.Config) > 0 {
					var configs []*sharedtypes.ConfigOption
					for key, value := range endpoint.Config {
						configs = append(configs, &sharedtypes.ConfigOption{
							Key:   sharedtypes.ConfigOptions_TIMEOUT, // Use TIMEOUT as it's the only available option
							Value: fmt.Sprintf("%s=%s", key, value),
						})
					}
					supplierEndpoint.Configs = configs
				}

				endpoints = append(endpoints, supplierEndpoint)
			}

			services = append(services, &sharedtypes.SupplierServiceConfig{
				ServiceId: svcConfig.ServiceId,
				Endpoints: endpoints,
			})
		}

		// Create the stake supplier message
		msg := suppliertypes.NewMsgStakeSupplier(
			supplier.OwnerAddress,
			addr.String(),
			supplier.Name,
			stakeAmount,
			services,
		)

		fmt.Printf("Staking supplier %s with %s...\n", supplier.Name, stakeAmount.String())

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
		err = cosmostx.Sign(ctx, txFactory, supplier.Name, txBuilder, true)
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

		fmt.Printf("Successfully staked supplier %s. Transaction hash: %s\n", supplier.Name, res.TxHash)

		// Sleep to avoid sequence issues
		time.Sleep(time.Second)
	}

	return nil
}

// Querier returns the querier instance for the staker
func (s *Staker) Querier() *Querier {
	return s.querier
}
