package token_logic_module

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

func getBlockProposer(ctx context.Context) (string, error) {
	return cosmostypes.AccAddress(cosmostypes.UnwrapSDKContext(ctx).BlockHeader().ProposerAddress).String(), nil
}
