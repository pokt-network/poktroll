package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) ClaimMorsePokt(goCtx context.Context, msg *types.MsgClaimMorsePokt) (*types.MsgClaimMorsePoktResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgClaimMorsePoktResponse{}, nil
}
