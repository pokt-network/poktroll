package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) ClaimMorseAccount(goCtx context.Context, msg *migrationtypes.MsgClaimMorseAccount) (*migrationtypes.MsgClaimMorseAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO_UPNEXT(@bryanchriswhite#1034): Handling the message
	_ = ctx

	return &migrationtypes.MsgClaimMorseAccountResponse{}, nil
}
