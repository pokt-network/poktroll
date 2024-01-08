package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := ms.Logger(ctx).With("method", "UpdateParams")

	if ms.authority != msg.Authority {
		logger.Error("invalid authority when updating params for the tokenomics module")
		return nil, fmt.Errorf(
			"expected authority account as only signer for proposal message; invalid authority; expected %s, got %s",
			ms.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	logger.Info("About to update params for the tokenomics module %v", msg.Params)
	ms.SetParams(ctx, msg.Params)
	logger.Info("Successfully updated params for the tokenomics module")

	return &types.MsgUpdateParamsResponse{}, nil
}
