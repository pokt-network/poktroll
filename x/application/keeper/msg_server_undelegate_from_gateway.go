package keeper

import (
	"context"
	"fmt"

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

	k.addPendingUndelegation(ctx, &types.Undelegation{
		AppAddress:     msg.AppAddress,
		GatewayAddress: msg.GatewayAddress,
	})

	isSuccessful = true
	return &types.MsgUndelegateFromGatewayResponse{}, nil
}

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

	hasPendingUndelegation := store.Has(types.PendingUndelegationsKey(pendingUndelegation))
	if !hasPendingUndelegation {
		store.Set(
			types.PendingUndelegationsKey(pendingUndelegation),
			k.cdc.MustMarshal(pendingUndelegation),
		)
	}
}
