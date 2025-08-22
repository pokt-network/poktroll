package token_logic_module

import (
	"context"
	"fmt"
	"math/big"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/pokt-network/poktroll/app/pocket"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// distributeSupplierRewardsToShareHolders distributes the supplier rewards to its
// shareholders based on the rev share percentage of the supplier service config.
func distributeSupplierRewardsToShareHolders(
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	settlementOpReason tokenomicstypes.SettlementOpReason,
	supplier *sharedtypes.Supplier,
	serviceId string,
	amountToDistribute math.Int,
) error {
	logger = logger.With(
		"method", "distributeSupplierRewardsToShareHolders",
		"session_id", result.GetSessionId(),
	)

	var serviceRevShares []*sharedtypes.ServiceRevenueShare
	for _, svc := range supplier.Services {
		if svc.ServiceId == serviceId {
			serviceRevShares = svc.RevShare
			break
		}
	}

	// This should theoretically never happen because the following validation
	// is done during staking: MsgStakeSupplier.ValidateBasic() -> ValidateSupplierServiceConfigs() -> ValidateServiceRevShare().
	// The check is here just for redundancy.
	if serviceRevShares == nil {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"SHOULD NEVER HAPPEN: service %q not found for supplier %v",
			serviceId,
			supplier,
		)
	}

	// NOTE: Use the serviceRevShares slice to iterate through the serviceRevSharesMap deterministically.
	shareAmountMap := GetShareAmountMap(serviceRevShares, amountToDistribute)
	for _, revShare := range serviceRevShares {
		shareAmount := shareAmountMap[revShare.GetAddress()]

		// Don't queue zero amount transfer operations.
		if shareAmount.IsZero() {
			// DEV_NOTE: This should never happen, but it mitigates a chain halt if it does.
			logger.Warn(fmt.Sprintf("zero shareAmount for service rev share address %q", revShare.GetAddress()))
			continue
		}

		// Queue the sending of the newley minted uPOKT from the supplier module
		// account to the supplier's shareholders.
		shareAmountCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, shareAmount)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         settlementOpReason,
			SenderModule:     suppliertypes.ModuleName,
			RecipientAddress: revShare.GetAddress(),
			Coin:             shareAmountCoin,
		})

		logger.Info(fmt.Sprintf("operation queued: send %s from the supplier module to the supplier shareholder with address %q", shareAmountCoin, supplier.GetOperatorAddress()))
	}

	logger.Info(fmt.Sprintf("operation queued: distribute %d uPOKT to supplier %q shareholders", amountToDistribute, supplier.GetOperatorAddress()))

	return nil
}

// GetShareAmountMap calculates the amount of uPOKT to distribute to each revenue
// shareholder based on the rev share percentage of the service.
// It returns a map of the shareholder address to the amount of uPOKT to distribute.
// The first shareholder gets any remainder resulting from the integer division.
// DEV_NOTE: It is publicly exposed to be used in the tests.
func GetShareAmountMap(
	serviceRevShare []*sharedtypes.ServiceRevenueShare,
	amountToDistribute math.Int,
) (shareAmountMap map[string]math.Int) {
	totalDistributed := math.NewInt(0)
	shareAmountMap = make(map[string]math.Int, len(serviceRevShare))

	for _, revShare := range serviceRevShare {
		sharePercentageRat := new(big.Rat).SetFrac64(int64(revShare.RevSharePercentage), 100)
		amountToDistributeRat := new(big.Rat).SetInt(amountToDistribute.BigInt())
		shareAmountRat := new(big.Rat).Mul(amountToDistributeRat, sharePercentageRat)
		shareAmountInt := new(big.Int).Quo(shareAmountRat.Num(), shareAmountRat.Denom())
		shareAmountMap[revShare.Address] = math.NewIntFromBigInt(shareAmountInt)

		totalDistributed = totalDistributed.Add(shareAmountMap[revShare.Address])
	}

	// Add any remainder to the first shareholder.
	remainder := amountToDistribute.Sub(totalDistributed)
	shareAmountMap[serviceRevShare[0].Address] = shareAmountMap[serviceRevShare[0].Address].Add(remainder)

	return shareAmountMap
}

// distributeValidatorRewardsToStakeholders distributes validator rewards directly to
// validators and their delegators using ModToAcctTransfer operations instead of
// the Cosmos SDK distribution module.
//
// This function:
// 1. Calculates validator commission based on their commission rate
// 2. Distributes remaining rewards proportionally to delegators based on their stake shares
// 3. Uses immediate distribution via ModToAcctTransfer instead of lazy reward tracking
//
// This approach provides architectural consistency with other tokenomics reward
// distributions and gives full control over the reward distribution logic.
func distributeValidatorRewardsToStakeholders(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validator stakingtypes.ValidatorI,
	validatorRewardAmount math.Int,
	validatorCommissionOpReason tokenomicstypes.SettlementOpReason,
	delegatorRewardOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeValidatorRewardsToStakeholders",
		"validator", validator.GetOperator(),
		"reward_amount", validatorRewardAmount,
	)

	if validatorRewardAmount.IsZero() {
		logger.Debug("validator reward amount is zero, skipping distribution")
		return nil
	}

	// 1. Calculate validator commission
	commissionRate := validator.GetCommission()
	commissionAmount := calculateValidatorCommission(validatorRewardAmount, commissionRate)

	// 2. Calculate remaining amount for delegator pool
	delegatorPoolAmount := validatorRewardAmount.Sub(commissionAmount)

	// 3. Distribute commission directly to validator if non-zero
	if !commissionAmount.IsZero() {
		// Convert validator operator address (poktvaloper...) to regular account address (pokt...)
		validatorAccAddr, err := getValidatorAccountAddress(validator.GetOperator())
		if err != nil {
			return fmt.Errorf("failed to convert validator operator address %s to account address: %w", validator.GetOperator(), err)
		}

		validatorCommissionCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, commissionAmount)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         validatorCommissionOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: validatorAccAddr,
			Coin:             validatorCommissionCoin,
		})
		logger.Debug(fmt.Sprintf("operation queued: validator commission (%v) to %s (account: %s)", validatorCommissionCoin, validator.GetOperator(), validatorAccAddr))
	}

	// 4. Distribute to delegators if there's a delegator pool
	if !delegatorPoolAmount.IsZero() {
		if err := distributeToDelegators(ctx, logger, result, stakingKeeper, validator, delegatorPoolAmount, validatorCommissionOpReason, delegatorRewardOpReason); err != nil {
			return fmt.Errorf("error distributing to delegators for validator %s: %w", validator.GetOperator(), err)
		}
	}

	// 5. Log total distribution
	totalDistributed := commissionAmount.Add(delegatorPoolAmount)
	if !totalDistributed.Equal(validatorRewardAmount) {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"validator reward distribution mismatch: expected %s, distributed %s (commission: %s, delegator pool: %s)",
			validatorRewardAmount, totalDistributed, commissionAmount, delegatorPoolAmount,
		)
	}

	logger.Info(fmt.Sprintf("successfully distributed (%v) to validator %s and delegators (commission: %v, delegators: %v)",
		cosmostypes.NewCoin(pocket.DenomuPOKT, validatorRewardAmount), validator.GetOperator(),
		cosmostypes.NewCoin(pocket.DenomuPOKT, commissionAmount),
		cosmostypes.NewCoin(pocket.DenomuPOKT, delegatorPoolAmount)))

	return nil
}

// distributeToDelegators distributes the delegator pool rewards to all delegators
// of a validator proportionally based on their stake shares.
func distributeToDelegators(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validator stakingtypes.ValidatorI,
	delegatorPoolAmount math.Int,
	validatorCommissionOpReason tokenomicstypes.SettlementOpReason,
	delegatorRewardOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeToDelegators",
		"validator", validator.GetOperator(),
		"pool_amount", delegatorPoolAmount,
	)

	// Get validator address for delegation queries
	valAddr, err := cosmostypes.ValAddressFromBech32(validator.GetOperator())
	if err != nil {
		return fmt.Errorf("invalid validator address %s: %w", validator.GetOperator(), err)
	}

	// Get all delegations for this validator
	delegations, err := stakingKeeper.GetValidatorDelegations(ctx, valAddr)
	if err != nil {
		return fmt.Errorf("failed to get delegations for validator %s: %w", validator.GetOperator(), err)
	}

	if len(delegations) == 0 {
		logger.Debug("no delegations found for validator, adding delegator pool to commission")

		// Convert validator operator address to regular account address
		validatorAccAddr, err := getValidatorAccountAddress(validator.GetOperator())
		if err != nil {
			return fmt.Errorf("failed to convert validator operator address %s to account address: %w", validator.GetOperator(), err)
		}

		// If no delegators, give the delegator pool amount back to the validator as commission
		additionalCommissionCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, delegatorPoolAmount)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         validatorCommissionOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: validatorAccAddr,
			Coin:             additionalCommissionCoin,
		})
		logger.Debug(fmt.Sprintf("operation queued: additional commission (%v) to validator %s (account: %s, no delegators)", additionalCommissionCoin, validator.GetOperator(), validatorAccAddr))
		return nil
	}

	// Get total delegator shares for proportional calculation
	totalShares := validator.GetDelegatorShares()
	if totalShares.IsNil() || totalShares.IsZero() {
		logger.Debug(fmt.Sprintf("validator %s has zero or nil delegator shares despite having delegations, giving delegator pool to validator as commission", validator.GetOperator()))

		// Convert validator operator address to regular account address
		validatorAccAddr, err := getValidatorAccountAddress(validator.GetOperator())
		if err != nil {
			return fmt.Errorf("failed to convert validator operator address %s to account address: %w", validator.GetOperator(), err)
		}

		// If delegator shares are invalid, give the delegator pool amount back to the validator as commission
		additionalCommissionCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, delegatorPoolAmount)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         validatorCommissionOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: validatorAccAddr,
			Coin:             additionalCommissionCoin,
		})
		logger.Debug(fmt.Sprintf("operation queued: delegator pool as additional commission (%v) to validator %s (account: %s, invalid delegator shares)", additionalCommissionCoin, validator.GetOperator(), validatorAccAddr))
		return nil
	}

	// Calculate and distribute rewards to each delegator
	delegatorSharesMap := calculateDelegatorShares(delegations, delegatorPoolAmount, totalShares)
	totalDistributedToDelegators := math.ZeroInt()

	for _, delegation := range delegations {
		delegatorShare := delegatorSharesMap[delegation.DelegatorAddress]

		if delegatorShare.IsZero() {
			logger.Debug(fmt.Sprintf("delegator %s calculated share is zero, skipping", delegation.DelegatorAddress))
			continue
		}

		// Queue the transfer to delegator
		delegatorCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, delegatorShare)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         delegatorRewardOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: delegation.DelegatorAddress,
			Coin:             delegatorCoin,
		})

		totalDistributedToDelegators = totalDistributedToDelegators.Add(delegatorShare)
		logger.Debug(fmt.Sprintf("operation queued: delegator reward (%v) to %s (shares: %s/%s)",
			delegatorCoin, delegation.DelegatorAddress, delegation.Shares, totalShares))
	}

	// Handle any remainder due to rounding - give it to validator as additional commission
	remainder := delegatorPoolAmount.Sub(totalDistributedToDelegators)
	if !remainder.IsZero() {
		// Convert validator operator address to regular account address
		validatorAccAddr, err := getValidatorAccountAddress(validator.GetOperator())
		if err != nil {
			return fmt.Errorf("failed to convert validator operator address %s to account address: %w", validator.GetOperator(), err)
		}

		remainderCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, remainder)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         validatorCommissionOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: validatorAccAddr,
			Coin:             remainderCoin,
		})
		logger.Debug(fmt.Sprintf("operation queued: remainder commission (%v) to validator %s (account: %s)", remainderCoin, validator.GetOperator(), validatorAccAddr))
	}

	logger.Info(fmt.Sprintf("distributed (%v) to %d delegators of validator %s (remainder to validator: %v)",
		cosmostypes.NewCoin(pocket.DenomuPOKT, totalDistributedToDelegators), len(delegations), validator.GetOperator(),
		cosmostypes.NewCoin(pocket.DenomuPOKT, remainder)))

	return nil
}

// calculateValidatorCommission calculates the commission amount for a validator
// based on their commission rate and the total reward amount.
func calculateValidatorCommission(totalReward math.Int, commissionRate math.LegacyDec) math.Int {
	if totalReward.IsZero() {
		return math.ZeroInt()
	}

	// Handle nil or zero commission rate
	if commissionRate.IsNil() || commissionRate.IsZero() {
		return math.ZeroInt()
	}

	// Convert commission rate from LegacyDec to big.Rat for precise calculation
	// LegacyDec uses 18 decimal places (1e18 precision)
	commissionRat := new(big.Rat).SetInt(commissionRate.BigInt())
	precisionRat := new(big.Rat).SetInt(math.NewInt(1e18).BigInt())
	commissionRat = commissionRat.Quo(commissionRat, precisionRat)
	rewardRat := new(big.Rat).SetInt(totalReward.BigInt())

	// Calculate commission: total_reward * commission_rate
	commissionAmountRat := new(big.Rat).Mul(rewardRat, commissionRat)

	// Convert back to Int, truncating (floor) any remainder
	commissionAmount := math.NewIntFromBigInt(new(big.Int).Quo(commissionAmountRat.Num(), commissionAmountRat.Denom()))

	return commissionAmount
}

// calculateDelegatorShares calculates the reward amount for each delegator
// based on their proportional stake shares. Similar to GetShareAmountMap
// but for validator delegations.
func calculateDelegatorShares(
	delegations []stakingtypes.Delegation,
	delegatorPoolAmount math.Int,
	totalShares math.LegacyDec,
) map[string]math.Int {
	shareAmountMap := make(map[string]math.Int, len(delegations))
	totalDistributed := math.ZeroInt()

	// Calculate proportional shares for each delegator
	for _, delegation := range delegations {
		// Calculate: (delegator_shares / total_shares) * pool_amount
		shareRatio := delegation.Shares.Quo(totalShares)
		poolAmountDec := math.LegacyNewDecFromInt(delegatorPoolAmount)
		delegatorAmount := poolAmountDec.Mul(shareRatio).TruncateInt()

		shareAmountMap[delegation.DelegatorAddress] = delegatorAmount
		totalDistributed = totalDistributed.Add(delegatorAmount)
	}

	// Add any remainder to the first delegator (similar to supplier distribution logic)
	if len(delegations) > 0 {
		remainder := delegatorPoolAmount.Sub(totalDistributed)
		if !remainder.IsZero() {
			firstDelegator := delegations[0].DelegatorAddress
			shareAmountMap[firstDelegator] = shareAmountMap[firstDelegator].Add(remainder)
		}
	}

	return shareAmountMap
}

// getValidatorAccountAddress converts a validator operator address (poktvaloper...)
// to a regular account address (pokt...) for coin transfers.
//
// In Cosmos SDK, validator operator addresses and account addresses have the same
// underlying bytes, just different bech32 prefixes.
func getValidatorAccountAddress(validatorOperatorAddr string) (string, error) {
	// Parse the validator operator address to get the underlying bytes
	valAddr, err := cosmostypes.ValAddressFromBech32(validatorOperatorAddr)
	if err != nil {
		return "", fmt.Errorf("invalid validator operator address %s: %w", validatorOperatorAddr, err)
	}

	// Convert validator address bytes to regular account address
	// Validator addresses and account addresses have the same underlying bytes
	accAddr := cosmostypes.AccAddress(valAddr.Bytes())

	return accAddr.String(), nil
}
