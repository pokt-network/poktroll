package keeper

import (
	"context"

    "github.com/pokt-network/poktroll/x/migration/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)


func (k msgServer) ImportMorseClaimableAccounts(goCtx context.Context,  msg *types.MsgImportMorseClaimableAccounts) (*types.MsgImportMorseClaimableAccountsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

    // TODO: Handling the message
    _ = ctx

	return &types.MsgImportMorseClaimableAccountsResponse{}, nil
}
