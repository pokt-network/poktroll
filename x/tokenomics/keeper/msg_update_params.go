package keeper

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
)

func (k msgServer) UpdateParams(ctx context.Context, msg *tokenomics.MsgUpdateParams) (*tokenomics.MsgUpdateParamsResponse, error) {
	logger := k.Logger()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != k.GetAuthority() {
		return nil, tokenomics.ErrTokenomicsInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			k.GetAuthority(),
			msg.Authority,
		)
	}

	logger.Info(fmt.Sprintf("About to update params from [%v] to [%v]", k.GetParams(ctx), msg.Params))

	if err := k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	logger.Info("Done updating params")

	return &tokenomics.MsgUpdateParamsResponse{}, nil
}

// ComputeUnitsToTokensMultiplier returns the ComputeUnitsToTokensMultiplier param
func (k Keeper) ComputeUnitsToTokensMultiplier(ctx context.Context) uint64 {
	return k.GetParams(ctx).ComputeUnitsToTokensMultiplier
}
