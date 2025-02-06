package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// CreateMorseAccountState creates the on-chain MorseAccountState ONLY ONCE (per network / re-genesis).
func (k msgServer) CreateMorseAccountState(goCtx context.Context, msg *types.MsgCreateMorseAccountState) (*types.MsgCreateMorseAccountStateResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value already exists
	_, isFound := k.GetMorseAccountState(ctx)
	if isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "already set")
	}

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	k.SetMorseAccountState(
		ctx,
		msg.MorseAccountState,
	)

	// TODO_UPNEXT(@bryanchriswhite#1034): Emit an event...

	// TODO_UPNEXT(@bryanchriswhite#1034): Populate the response...
	return &types.MsgCreateMorseAccountStateResponse{}, nil
}
