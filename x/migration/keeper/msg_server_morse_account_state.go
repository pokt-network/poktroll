package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// CreateMorseAccountState creates the on-chain MorseAccountState ONLY ONCE (per network / re-genesis).
func (k msgServer) CreateMorseAccountState(
	ctx context.Context,
	msg *migrationtypes.MsgCreateMorseAccountState,
) (*migrationtypes.MsgCreateMorseAccountStateResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("method", "CreateMorseAccountState")

	if err := msg.ValidateBasic(); err != nil {
		logger.Info(err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if the value already exists
	if _, isFound := k.GetMorseAccountState(sdkCtx); isFound {
		err := migrationtypes.ErrMorseAccountState.Wrap("already set")
		logger.Info(err.Error())
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	k.SetMorseAccountState(sdkCtx, msg.MorseAccountState)

	stateHash, err := msg.MorseAccountState.GetHash()
	if err != nil {
		logger.Info(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err = sdkCtx.EventManager().EmitTypedEvent(
		&migrationtypes.EventCreateMorseAccountState{
			CreatedAtHeight:       sdkCtx.BlockHeight(),
			MorseAccountStateHash: stateHash,
		},
	); err != nil {
		logger.Info(err.Error())
		return nil, err
	}

	return &migrationtypes.MsgCreateMorseAccountStateResponse{
		StateHash:   stateHash,
		NumAccounts: uint64(len(msg.MorseAccountState.Accounts)),
	}, nil
}
