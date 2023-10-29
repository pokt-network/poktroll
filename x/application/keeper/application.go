package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"pocket/x/application/types"
)

// SetApplication set a specific application in the store from its index
func (k Keeper) SetApplication(ctx sdk.Context, application types.Application) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))
	b := k.cdc.MustMarshal(&application)
	store.Set(types.ApplicationKey(
		application.Address,
	), b)
}

// GetApplication returns a application from its index
func (k Keeper) GetApplication(
	ctx sdk.Context,
	appAddr string,
) (app types.Application, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))

	b := store.Get(types.ApplicationKey(
		appAddr,
	))
	if b == nil {
		return app, false
	}

	k.cdc.MustUnmarshal(b, &app)
	return app, true
}

// RemoveApplication removes a application from the store
func (k Keeper) RemoveApplication(
	ctx sdk.Context,
	appAddr string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))
	store.Delete(types.ApplicationKey(
		appAddr,
	))
}

// GetAllApplication returns all application
func (k Keeper) GetAllApplication(ctx sdk.Context) (apps []types.Application) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Application
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		apps = append(apps, val)
	}

	return
}

// UndelegateGateway undelegates the specified gateway for the application address
func (k Keeper) UndelegateGateway(ctx sdk.Context, appAddress, gatewayAddress string) error {
	logger := k.Logger(ctx).With("method", "UndelegateFromGateway")

	// Retrieve the application from the store
	app, found := k.GetApplication(ctx, appAddress)
	if !found {
		logger.Info("Application not found with address [%s]", appAddress)
		return sdkerrors.Wrapf(types.ErrAppNotFound, "application not found with address: %s", appAddress)
	}
	logger.Info("Application found with address [%s]", appAddress)

	// Check if the application is already delegated to the gateway
	foundIdx := -1
	for i, gatewayAddr := range app.DelegateeGatewayAddresses {
		if gatewayAddr == gatewayAddress {
			foundIdx = i
		}
	}
	if foundIdx == -1 {
		logger.Info("Application not delegated to gateway with address [%s]", gatewayAddress)
		return sdkerrors.Wrapf(types.ErrAppNotDelegated, "application not delegated to gateway with address: %s", gatewayAddress)
	}

	// Remove the gateway from the application's delegatee gateway public keys
	app.DelegateeGatewayAddresses = append(app.DelegateeGatewayAddresses[:foundIdx], app.DelegateeGatewayAddresses[foundIdx+1:]...)

	// Update the application store with the new delegation
	k.SetApplication(ctx, app)

	return nil
}
