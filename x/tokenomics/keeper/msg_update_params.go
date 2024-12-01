package keeper

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

func (k msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	logger := k.Logger()

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != k.GetAuthority() {
		return nil, types.ErrTokenomicsInvalidSigner.Wrapf(
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

	return &types.MsgUpdateParamsResponse{
		Params: &msg.Params,
	}, nil
}
