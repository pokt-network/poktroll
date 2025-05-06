package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) RecoverMorseAccount(ctx context.Context, msg *types.MsgRecoverMorseAccount) (*types.MsgRecoverMorseAccountResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// TODO_MAINNET_MIGRATION(@bryanchriswhite): Implement MsgRecoverMorseAccount handler...
	_ = sdkCtx

	return &types.MsgRecoverMorseAccountResponse{}, nil
}
