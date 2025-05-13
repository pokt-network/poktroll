package keeper

import (
	"context"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) ClaimMorseMultiSigAccount(ctx context.Context, msg *migrationtypes.MsgClaimMorseMultiSigAccount) (*migrationtypes.MsgClaimMorseAccountResponse, error) {
	res, err := k.ClaimMorseAccountMessage(ctx, msg)
	return res, err
}
