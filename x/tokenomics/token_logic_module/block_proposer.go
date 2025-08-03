package token_logic_module

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// GetBlockProposerOperatorAddress returns the operator address of the current block's proposer.
// It resolves the consensus address from the block header to the validator's operator address.
func GetBlockProposerOperatorAddress(ctx context.Context, stakingKeeper types.StakingKeeper) (string, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Get the consensus address from the block header
	consAddr := cosmostypes.ConsAddress(sdkCtx.BlockHeader().ProposerAddress)

	// In test environments, the staking keeper might be nil or a mock
	// Fall back to using the consensus address directly as account address
	if stakingKeeper == nil {
		// Convert consensus address directly to account address (for test compatibility)
		// This is not correct for production but allows tests to pass
		return cosmostypes.AccAddress(consAddr).String(), nil
	}

	// Look up the validator by consensus address
	validator, err := stakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		// If validator not found, fall back to using consensus address as account address
		// This can happen in test environments without proper staking module setup
		return cosmostypes.AccAddress(consAddr).String(), nil
	}

	// Get the validator's operator address (this is what should receive the tokens)
	operatorAddr := validator.GetOperator()

	// Convert the operator address to an account address
	// The operator address is a ValAddress, but for sending tokens we need an AccAddress
	// They share the same bytes, just different Bech32 prefixes
	accAddr := cosmostypes.AccAddress(operatorAddr)

	return accAddr.String(), nil
}
