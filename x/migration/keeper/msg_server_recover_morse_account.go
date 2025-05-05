package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) RecoverMorseAccount(goCtx context.Context, msg *types.MsgRecoverMorseAccount) (*types.MsgRecoverMorseAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgRecoverMorseAccountResponse{}, nil
}
