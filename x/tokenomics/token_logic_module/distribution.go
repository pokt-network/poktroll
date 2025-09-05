package token_logic_module

import (
	"context"
	"fmt"
	"math/big"
	"sort"

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

// distributeValidatorRewards distributes session settlement rewards to all bonded validators.
//   - Rewards are distributed proportionally based on their staking weight regardless of who the proposer is.
//   - This implements pure stake-weighted distribution without any commission calculations, as these are session
//     settlement rewards (not consensus rewards).
//
// The distribution formula is:
//
//	validatorReward = totalValidatorRewardAmount × (validatorStake / totalBondedStake)
func distributeValidatorRewards(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	totalValidatorRewardAmount math.Int,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeValidatorRewards",
		"session_id", result.GetSessionId(),
		"total_reward_amount", totalValidatorRewardAmount,
	)

	// Phase 1: Validate inputs and prepare validator data
	validators, totalBondedTokens, err := validateAndPrepareValidatorRewards(ctx, logger, stakingKeeper, totalValidatorRewardAmount)
	if err != nil {
		return err
	}
	if validators == nil {
		// Skip distribution (zero amount, no validators, etc.)
		return nil
	}

	// Phase 2: Calculate base rewards and fractional remainders
	rewards, fractions, totalAssigned := calculateBaseValidatorRewards(
		logger,
		validators,
		totalBondedTokens,
		totalValidatorRewardAmount,
	)

	// Phase 3: Distribute remainder using Largest Remainder Method
	distributeRemainder(logger, rewards, fractions, totalAssigned, totalValidatorRewardAmount)

	// Phase 4: Execute validator transfers
	return executeValidatorTransfers(
		logger,
		result,
		validators,
		rewards,
		totalBondedTokens,
		totalValidatorRewardAmount,
		settlementOpReason,
	)
}

// validateAndPrepareValidatorRewards performs input validation and prepares validator data for distribution.
// Returns validators and totalBondedTokens, or (nil, zero) if distribution should be skipped.
func validateAndPrepareValidatorRewards(
	ctx context.Context,
	logger cosmoslog.Logger,
	stakingKeeper tokenomicstypes.StakingKeeper,
	totalValidatorRewardAmount math.Int,
) ([]stakingtypes.Validator, math.Int, error) {
	// Should theoretically never happen
	if totalValidatorRewardAmount.IsZero() {
		logger.Debug("validator reward amount is zero, skipping distribution")
		return nil, math.ZeroInt(), nil
	}

	// Get all bonded validators
	validators, err := stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return nil, math.ZeroInt(), tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"failed to get bonded validators: %v", err,
		)
	}
	if len(validators) == 0 {
		logger.Warn("no bonded validators found, skipping validator reward distribution")
		return nil, math.ZeroInt(), nil
	}

	// Calculate total bonded tokens across all validators
	totalBondedTokens := math.ZeroInt()
	for _, validator := range validators {
		totalBondedTokens = totalBondedTokens.Add(validator.GetBondedTokens())
	}
	if totalBondedTokens.IsZero() {
		logger.Warn("total bonded tokens is zero, skipping validator reward distribution")
		return nil, math.ZeroInt(), nil
	}

	logger.Info(fmt.Sprintf(
		"distributing %s to %d validators based on stake weight (total bonded: %s)",
		totalValidatorRewardAmount.String(),
		len(validators),
		totalBondedTokens.String(),
	))

	return validators, totalBondedTokens, nil
}

// validatorFraction tracks fractional remainders for the Largest Remainder Method.
type validatorFraction struct {
	index    int
	fraction *big.Rat
}

// calculateBaseValidatorRewards implements Phase 1 of the Largest Remainder Method.
// Calculates base integer rewards for each validator and collects fractional remainders.
func calculateBaseValidatorRewards(
	logger cosmoslog.Logger,
	validators []stakingtypes.Validator,
	totalBondedTokens math.Int,
	totalValidatorRewardAmount math.Int,
) ([]math.Int, []validatorFraction, math.Int) {
	validatorRewards := make([]math.Int, len(validators))
	totalValidatorRewardsDistributed := math.ZeroInt()
	fractions := make([]validatorFraction, 0, len(validators))

	for i, validator := range validators {
		validatorStake := validator.GetBondedTokens()

		// Calculate exact proportional reward using big.Rat for determinacy
		exactRatio := new(big.Rat).SetFrac(
			new(big.Int).Mul(validatorStake.BigInt(), totalValidatorRewardAmount.BigInt()),
			totalBondedTokens.BigInt(),
		)

		// Split into integer (base reward) and fractional parts
		baseReward := new(big.Int).Quo(exactRatio.Num(), exactRatio.Denom())
		validatorRewards[i] = math.NewIntFromBigInt(baseReward)
		totalValidatorRewardsDistributed = totalValidatorRewardsDistributed.Add(validatorRewards[i])

		// Calculate fractional remainder using big.Rat for precise comparison
		baseRat := new(big.Rat).SetInt(baseReward)
		fractionalPart := new(big.Rat).Sub(exactRatio, baseRat)

		// Only store non-zero fractions to optimize sorting performance
		if fractionalPart.Sign() > 0 {
			fractions = append(fractions, validatorFraction{i, fractionalPart})
		}

		logger.Debug(fmt.Sprintf("  Validator %d (%s): stake=%s, base_reward=%s, fraction=%s",
			i, validator.GetOperator(), validatorStake.String(), validatorRewards[i].String(), fractionalPart.FloatString(6)))
	}

	return validatorRewards, fractions, totalValidatorRewardsDistributed
}

// distributeRemainder implements Phase 2 of the Largest Remainder Method.
// Distributes remainder tokens to validators with the largest fractional remainders.
func distributeRemainder(
	logger cosmoslog.Logger,
	rewards []math.Int,
	fractions []validatorFraction,
	totalAssigned math.Int,
	totalValidatorRewardAmount math.Int,
) {
	validatorRewardsRemainder := totalValidatorRewardAmount.Sub(totalAssigned)
	remainderTokensToDistribute := validatorRewardsRemainder.Int64()

	logger.Debug(fmt.Sprintf("Total calculated: %s, target: %s, remainder: %s tokens",
		totalAssigned.String(), totalValidatorRewardAmount.String(), validatorRewardsRemainder.String()))

	if remainderTokensToDistribute > 0 && len(fractions) > 0 {
		// Sort validators by fractional remainder (descending) using deterministic big.Rat comparison
		sort.Slice(fractions, func(i, j int) bool {
			return fractions[i].fraction.Cmp(fractions[j].fraction) > 0
		})

		logger.Debug(fmt.Sprintf("Distributing %d remainder tokens using Largest Remainder Method", remainderTokensToDistribute))

		// Distribute remainder tokens to validators with largest fractional remainders
		for t := range remainderTokensToDistribute {
			// Use modulo to cycle through validators if remainder > number of validators with fractions
			idx := fractions[t%int64(len(fractions))].index
			rewards[idx] = rewards[idx].AddRaw(1)

			logger.Debug(fmt.Sprintf("  Added 1 token to validator %d (fraction: %s)",
				idx, fractions[t%int64(len(fractions))].fraction.FloatString(6)))
		}
	} else if remainderTokensToDistribute > 0 {
		// Edge case: remainder exists but no fractional parts (shouldn't happen with proper math)
		logger.Warn(fmt.Sprintf("Rare edge case: remainder %d tokens with no fractional parts - adding to first validator", remainderTokensToDistribute))
		rewards[0] = rewards[0].Add(validatorRewardsRemainder)
	}
}

// executeValidatorTransfers converts validator addresses and queues reward transfer operations.
func executeValidatorTransfers(
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	validators []stakingtypes.Validator,
	rewards []math.Int,
	totalBondedTokens math.Int,
	totalValidatorRewardAmount math.Int,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	totalDistributed := math.ZeroInt()
	for i, validator := range validators {
		validatorReward := rewards[i]
		totalDistributed = totalDistributed.Add(validatorReward)

		if validatorReward.IsZero() {
			logger.Debug(fmt.Sprintf(
				"validator %s reward is zero, skipping",
				validator.GetOperator(),
			))
			continue
		}

		// Get validator account address for reward transfer
		// Convert validator operator address to account address
		valAddr, err := cosmostypes.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			logger.Error(fmt.Sprintf(
				"failed to parse validator operator address %s: %v",
				validator.GetOperator(), err,
			))
			continue
		}
		// Convert validator address to account address
		validatorAccAddr := cosmostypes.AccAddress(valAddr)

		// Queue the reward transfer to the validator
		validatorRewardCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, validatorReward)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         settlementOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: validatorAccAddr.String(),
			Coin:             validatorRewardCoin,
		})

		logger.Debug(fmt.Sprintf(
			"queued reward transfer: %s to validator %s (stake: %s, share: %s%%)",
			validatorRewardCoin.String(),
			validator.GetOperator(),
			validator.GetBondedTokens().String(),
			new(big.Rat).SetFrac(
				validator.GetBondedTokens().BigInt(),
				totalBondedTokens.BigInt(),
			).FloatString(2),
		))
	}

	logger.Info(fmt.Sprintf(
		"validator reward distribution complete: distributed %s of %s uPOKT to %d validators",
		totalDistributed.String(),
		totalValidatorRewardAmount.String(),
		len(validators),
	))

	return nil
}
