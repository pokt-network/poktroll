package keeper

import (
	"context"
	"fmt"

	"pocket/x/application/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) StakeApplication(
	goCtx context.Context,
	msg *types.MsgStakeApplication,
) (*types.MsgStakeApplicationResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	logger := k.Logger(ctx).With("method", "StakeApplication")
	logger.Info(fmt.Sprintf("About to stake application with msg: %v", msg))

	// Check if the value exists
	app, isAppFound := k.GetApplication(ctx, msg.Address)
	if !isAppFound {
		k.createApplication(ctx, msg)
	} else {
		k.updateApplication(ctx, app, types.Application)
	}

	// Checks if the the msg address is the same as the current owner
	if msg.Address != valFound.Address {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	var testapp = types.Testapp{
		Address: msg.Address,
		Index:   msg.Index,
	}

	k.SetApplication(ctx, app)
	logger.Info(fmt.Sprintf("successfully updated application stake: %v", app))

	return &types.MsgStakeApplicationResponse{}, nil
}

func (k msgServer) createApplication(
	ctx sdk.Context,
	msg *types.MsgStakeApplication,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKey))
	b := k.cdc.MustMarshalBinaryBare(&app)
	store.Set(types.KeyPrefix(types.ApplicationKey), b)
}

func (k msgServer) updateApplication(
	ctx sdk.Context,
	app types.Application,
	msg *types.MsgStakeApplication,
) {
	// Checks if the the msg address is the same as the current owner
	if msg.Address != app.Address.Address {
		return nil, cos.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.ApplicationKey))
	b := k.cdc.MustMarshalBinaryBare(&app)
	store.Set(types.KeyPrefix(types.ApplicationKey), b)
}
