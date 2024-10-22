package keeper

import (
	"context"
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

func (k Keeper) AddDelegation(ctx context.Context, gatewayAddr, AppAddr string) error {
	gateway, found := k.GetGateway(ctx, gatewayAddr)
	if !found {
		return types.ErrGatewayNotFound
	}
	if !slices.Contains(gateway.DelegatingApplicationAddresses, AppAddr) {
		gateway.DelegatingApplicationAddresses = append(gateway.DelegatingApplicationAddresses, AppAddr)
		k.SetGateway(ctx, gateway)
	}

	return nil
}

func (k Keeper) RemoveDelegation(ctx context.Context, gatewayAddr, AppAddr string) error {
	gateway, found := k.GetGateway(ctx, gatewayAddr)
	if !found {
		return types.ErrGatewayNotFound
	}

	if idx := slices.Index(gateway.DelegatingApplicationAddresses, AppAddr); idx >= 0 {
		gateway.DelegatingApplicationAddresses = append(
			gateway.DelegatingApplicationAddresses[:idx],
			gateway.DelegatingApplicationAddresses[idx+1:]...,
		)
		k.SetGateway(ctx, gateway)
	}

	return nil
}

func (k Keeper) EndBlockerUnbondGateways(ctx context.Context) error {
	logger := k.Logger().With("method", "EndBlockerUnbondGateways")
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	gatewaysToUnbond := make([]*types.Gateway, 0)
	for _, gateway := range k.GetAllGateways(ctx) {
		if gateway.UnstakeSessionEndHeight > sdkCtx.BlockHeight() {
			continue
		}
		gatewaysToUnbond = append(gatewaysToUnbond, &gateway)
		gatewayAddress, err := sdk.AccAddressFromBech32(gateway.Address)
		if err != nil {
			return err
		}
		// Send the coins from the gateway pool back to the gateway

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, gatewayAddress, []sdk.Coin{*gateway.Stake})
		if err != nil {
			logger.Error(fmt.Sprintf("could not send %v coins from %s module to %s account due to %v", gateway.Stake, gatewayAddress, types.ModuleName, err))
			return err
		}

	}

	for _, gateway := range gatewaysToUnbond {
		// Update the Gateway in the store
		k.RemoveGateway(ctx, gateway.Address)
		logger.Info(fmt.Sprintf("Successfully removed the gateway: %+v", gateway))
	}

	return nil
}
