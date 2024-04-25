package keeper

import (
	"context"
	"fmt"
	"slices"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/application/types"
)

func (k msgServer) UndelegateFromGateway(ctx context.Context, msg *types.MsgUndelegateFromGateway) (*types.MsgUndelegateFromGatewayResponse, error) {
	isSuccessful := false
	defer telemetry.EventSuccessCounter(
		"undelegate_from_gateway",
		telemetry.DefaultCounterFn,
		func() bool { return isSuccessful },
	)

	logger := k.Logger().With("method", "UndelegateFromGateway")
	logger.Info(fmt.Sprintf("About to schedule undelegate application from gateway with msg: %v", msg))

	if err := msg.ValidateBasic(); err != nil {
		logger.Error(fmt.Sprintf("Undelegation Message failed basic validation: %v", err))
		return nil, err
	}

	// Retrieve the application from the store
	foundApp, isAppFound := k.GetApplication(ctx, msg.AppAddress)
	if !isAppFound {
		return nil, types.ErrAppNotFound.Wrapf(
			"application not found with address %q",
			msg.AppAddress,
		)
	}

	// Check if the application is already delegated to the gateway
	gatewayIdx := slices.Index(foundApp.DelegateeGatewayAddresses, msg.GatewayAddress)
	if gatewayIdx == -1 {
		return nil, types.ErrAppNotDelegated.Wrapf(
			"application not delegated to gateway with address %q",
			msg.GatewayAddress,
		)
	}

	// The requested undelegation is not immediate, but scheduled to be active
	// at the start of the next session.
	k.addPendingUndelegation(ctx, &types.Undelegation{
		AppAddress:     msg.AppAddress,
		GatewayAddress: msg.GatewayAddress,
	})

	isSuccessful = true
	return &types.MsgUndelegateFromGatewayResponse{}, nil
}

// addPendingUndelegation adds a undelegation to the pending undelegations store.
// The undelegation will be then processed by EndBlockerProcessPendingUndelegations
// at the end of the session.
func (k Keeper) addPendingUndelegation(
	ctx context.Context,
	pendingUndelegation *types.Undelegation,
) {
	k.Logger().With("method", "addPendingUndelegation").
		Info(fmt.Sprintf("Adding pending undelegation %v", pendingUndelegation))

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	store := prefix.NewStore(
		storeAdapter,
		types.KeyPrefix(types.PendingUndelegationsKeyPrefix),
	)

	// If the undelegation is already in the pending undelegations store, do not add it again.
	if !store.Has(types.PendingUndelegationKey(pendingUndelegation)) {
		pendingUndelegationKey := types.PendingUndelegationKey(pendingUndelegation)
		pendingUndelegationBz := k.cdc.MustMarshal(pendingUndelegation)
		store.Set(pendingUndelegationKey, pendingUndelegationBz)
	}
}
