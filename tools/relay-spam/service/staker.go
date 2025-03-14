package service

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
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
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

	// Create a new service querier
	querier, err := NewQuerier(clientConn)
	if err != nil {
		// Log the error but continue without the querier
		fmt.Printf("Failed to create service querier: %v\n", err)
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

// AddService adds a single service to the blockchain
func (s *Staker) AddService(svc config.Service) error {
	ctx := context.Background()

	// Check if the service already exists
	serviceExists := false
	if s.querier != nil {
		var err error
		serviceExists, err = s.querier.ServiceExists(ctx, svc.ServiceId)
		if err != nil {
			// If we get an error checking if the service exists, log it but continue
			fmt.Printf("Warning: Error checking if service %s exists: %v\n", svc.Name, err)
			fmt.Printf("Assuming service %s does not exist, proceeding with adding\n", svc.Name)
			serviceExists = false
		} else if serviceExists {
			fmt.Printf("Service %s with ID %s already exists, skipping\n", svc.Name, svc.ServiceId)
			return nil
		}
	} else {
		fmt.Printf("Querier not available, assuming service %s does not exist\n", svc.Name)
	}

	// Get account from keyring
	key, err := s.clientCtx.Keyring.Key(svc.Name)
	if err != nil {
		return fmt.Errorf("failed to get key for %s: %w", svc.Name, err)
	}

	// Get the address from the key
	addr, err := key.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get address for %s: %w", svc.Name, err)
	}
	fmt.Printf("Using address %s for service %s\n", addr.String(), svc.Name)

	// Create the add service message
	msg := servicetypes.NewMsgAddService(
		addr.String(),
		svc.ServiceId,
		svc.ServiceName,
		servicetypes.DefaultComputeUnitsPerRelay, // Using default compute units per relay
	)

	fmt.Printf("Adding service %s with ID %s...\n", svc.Name, svc.ServiceId)

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
	err = cosmostx.Sign(ctx, txFactory, svc.Name, txBuilder, true)
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

	fmt.Printf("Successfully added service %s with ID %s. Transaction hash: %s\n", svc.Name, svc.ServiceId, res.TxHash)

	// Sleep to avoid sequence issues
	time.Sleep(time.Second)

	return nil
}

// AddServices adds all services that are not already added on the blockchain
func (s *Staker) AddServices() error {
	ctx := context.Background()

	for _, svc := range s.config.Services {
		// Check if the service already exists
		serviceExists := false
		if s.querier != nil {
			var err error
			serviceExists, err = s.querier.ServiceExists(ctx, svc.ServiceId)
			if err != nil {
				// If we get an error checking if the service exists, log it but continue
				fmt.Printf("Warning: Error checking if service %s exists: %v\n", svc.Name, err)
				fmt.Printf("Assuming service %s does not exist, proceeding with adding\n", svc.Name)
				serviceExists = false
			} else if serviceExists {
				fmt.Printf("Service %s with ID %s already exists, skipping\n", svc.Name, svc.ServiceId)
				continue
			}
		} else {
			fmt.Printf("Querier not available, assuming service %s does not exist\n", svc.Name)
		}

		// Get account from keyring
		key, err := s.clientCtx.Keyring.Key(svc.Name)
		if err != nil {
			return fmt.Errorf("failed to get key for %s: %w", svc.Name, err)
		}

		// Get the address from the key
		addr, err := key.GetAddress()
		if err != nil {
			return fmt.Errorf("failed to get address for %s: %w", svc.Name, err)
		}
		fmt.Printf("Using address %s for service %s\n", addr.String(), svc.Name)

		// Create the add service message
		msg := servicetypes.NewMsgAddService(
			addr.String(),
			svc.ServiceId,
			svc.ServiceName,
			servicetypes.DefaultComputeUnitsPerRelay, // Using default compute units per relay
		)

		fmt.Printf("Adding service %s with ID %s...\n", svc.Name, svc.ServiceId)

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
		err = cosmostx.Sign(ctx, txFactory, svc.Name, txBuilder, true)
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

		fmt.Printf("Successfully added service %s with ID %s. Transaction hash: %s\n", svc.Name, svc.ServiceId, res.TxHash)

		// Sleep to avoid sequence issues
		time.Sleep(time.Second)
	}

	return nil
}

// Querier returns the querier instance for the staker
func (s *Staker) Querier() *Querier {
	return s.querier
}
