package token_logic_module

import (
	"context"
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// GetBlockProposerOperatorAddress returns the operator address of the current block's proposer.
// It resolves the consensus address from the block header to the validator's operator address.
func GetBlockProposerOperatorAddress(ctx context.Context, stakingKeeper types.StakingKeeper) (string, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Get the consensus address from the block header
	consAddr := cosmostypes.ConsAddress(sdkCtx.BlockHeader().ProposerAddress)

	// Look up the validator by consensus address
	validator, err := stakingKeeper.ValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		return "", fmt.Errorf("failed to get validator by consensus address %s: %w", consAddr.String(), err)
	}

	if validator == nil {
		return "", fmt.Errorf("validator not found for consensus address %s", consAddr.String())
	}

	// Get the validator's operator address (this is what should receive the tokens)
	operatorAddr := validator.GetOperator()

	// Convert the operator address to an account address
	// The operator address is a ValAddress, but for sending tokens we need an AccAddress
	// They share the same bytes, just different Bech32 prefixes
	accAddr := cosmostypes.AccAddress(operatorAddr)

	return accAddr.String(), nil
}
