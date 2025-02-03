package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/x/migration/types"
)

// CreateMorseAccountState creates the on-chain MorseAccountState ONLY ONCE (per network / re-genesis).
func (k msgServer) CreateMorseAccountState(ctx context.Context, msg *types.MsgCreateMorseAccountState) (*types.MsgCreateMorseAccountStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if the value already exists
	if _, isFound := k.GetMorseAccountState(sdkCtx); isFound {
		return nil, status.Error(
			codes.FailedPrecondition,
			sdkerrors.ErrInvalidRequest.Wrap("already set").Error(),
		)
	}

	k.SetMorseAccountState(
		sdkCtx,
		msg.MorseAccountState,
	)

	stateHash, err := msg.MorseAccountState.GetHash()
	if err != nil {
		return nil, status.Error(
			codes.Internal,
			err.Error(),
		)
	}

	if err = sdkCtx.EventManager().EmitTypedEvent(
		&types.EventCreateMorseAccountState{
			Height:    sdkCtx.BlockHeight(),
			StateHash: stateHash,
		},
	); err != nil {
		return nil, err
	}

	return &types.MsgCreateMorseAccountStateResponse{
		StateHash:   stateHash,
		NumAccounts: uint64(len(msg.MorseAccountState.Accounts)),
	}, nil
}
