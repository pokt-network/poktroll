package token_logic_module

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// getBlockProposerOperatorAddress returns the operator address of the current block's proposer.
// It resolves the consensus address from the block header to the validator's operator address.
func getBlockProposerOperatorAddress(
	ctx context.Context,
	stakingKeeper types.StakingKeeper,
) (string, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Get the consensus address from the block header
	consAddr := cosmostypes.ConsAddress(sdkCtx.BlockHeader().ProposerAddress)

	// Look up the validator by consensus address
	validator, err := stakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	if err != nil {
		return "", err
	}

	// Get the validator's operator address (this is what should receive the tokens)
	operatorAddrString := validator.GetOperator()

	// Parse the validator operator address
	valAddr, err := cosmostypes.ValAddressFromBech32(operatorAddrString)
	if err != nil {
		return "", err
	}
	
	// Convert the operator address to an account address.
	// The operator address is a ValAddress, but for sending tokens we need an AccAddress
	// They share the same bytes, just different Bech32 prefixes
	accAddr := cosmostypes.AccAddress(valAddr)

	return accAddr.String(), nil
}
