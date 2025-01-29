package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) CreateMorseAccountState(goCtx context.Context, msg *types.MsgCreateMorseAccountState) (*types.MsgCreateMorseAccountStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value already exists
	_, isFound := k.GetMorseAccountState(ctx)
	if isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "already set")
	}

	k.SetMorseAccountState(
		ctx,
		msg.MorseAccountState,
	)
	return &types.MsgCreateMorseAccountStateResponse{}, nil
}
