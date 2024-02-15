package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func (k msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Logger()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// TODO_BLOCKER(@Olshansk): How do we validate this is the same address that signed the request?
	// Do we have to use `msg.GetSigners()` explicitly during the check/validation or
	// does the `cosmos.msg.v1.signer` tag in the protobuf definition enforce
	// this somewhere in the Cosmos SDK?
	if msg.Authority != k.GetAuthority() {
		return nil, types.ErrTokenomicsAuthorityAddressMismatch
	}

	prevParams := k.GetParams(ctx)
	logger.Info("About to update params from [%v] to [%v]", prevParams, msg.Params)

	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	logger.Info("Done updating params")

	return &types.MsgUpdateParamsResponse{}, nil
}

// ComputeUnitsToTokensMultiplier returns the ComputeUnitsToTokensMultiplier param
func (k Keeper) ComputeUnitsToTokensMultiplier(ctx context.Context) (param uint64) {
	return k.GetParams(ctx).ComputeUnitsToTokensMultiplier
}
