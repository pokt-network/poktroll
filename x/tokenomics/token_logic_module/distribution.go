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

// distributeValidatorRewards distributes session settlement rewards to all bonded validators
// and their delegators.
//   - Rewards are distributed proportionally based on their staking weight regardless of who the proposer is.
//   - This implements pure stake-weighted distribution without any commission calculations, as these are session
//     settlement rewards (not consensus rewards).
//   - Each validator's total bonded tokens includes both self-bonded and delegated stakes.
//   - Delegators receive rewards proportional to their delegated stake.
//
// The distribution formula is:
//
//	stakeholderReward = totalValidatorRewardAmount × (stakeholderStake / totalBondedStake)
//
// Where stakeholders include both validators (self-bonded stake) and all delegators.
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

	// Phase 2: Distribute rewards to validators and their delegators
	return distributeToValidatorsAndDelegators(
		ctx,
		logger,
		result,
		stakingKeeper,
		validators,
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

// distributeToValidatorsAndDelegators is a helper function that distributes rewards to
// validators and their delegators based purely on stake proportions. This function
// implements proportional stake-based distribution without any commission calculations.
//
// The distribution formula is:
// stakeholderReward = totalValidatorReward × (stakeholderStake / totalBondedStake)
//
// Where stakeholders include both the validator (self-bonded stake) and all delegators
// who have delegated to that validator.
func distributeToValidatorsAndDelegators(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validators []stakingtypes.Validator,
	totalBondedTokens math.Int,
	validatorRewardAmount math.Int,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeToValidatorsAndDelegators",
		"session_id", result.GetSessionId(),
		"total_reward_amount", validatorRewardAmount,
	)

	// Input validation is already done in the main distributeValidatorRewards function

	logger.Info(fmt.Sprintf(
		"distributing %s to validators and delegators based on stake weight (total bonded: %s)",
		validatorRewardAmount.String(),
		totalBondedTokens.String(),
	))

	// Track stakes for each recipient address during discovery phase
	stakeAmounts := make(map[string]math.Int)

	// Process each validator and their delegators
	for _, validator := range validators {
		valAddr, err := cosmostypes.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			logger.Error(fmt.Sprintf(
				"failed to parse validator operator address %s: %v",
				validator.GetOperator(), err,
			))
			continue
		}

		validatorAccAddr := cosmostypes.AccAddress(valAddr)

		// Try to get delegations to understand the true stake breakdown
		delegations, err := stakingKeeper.GetValidatorDelegations(ctx, valAddr)
		if err != nil {
			// On delegation query error, treat the entire validator bonded tokens as validator-only rewards
			// This maintains backward compatibility and ensures validators still receive rewards despite delegation query failures
			logger.Error(fmt.Sprintf(
				"failed to get delegations for validator %s: %v (using validator-only distribution)",
				validator.GetOperator(), err,
			))

			validatorBondedTokens := validator.GetBondedTokens()
			if !validatorBondedTokens.IsZero() {
				validatorAddrStr := validatorAccAddr.String()
				stakeAmounts[validatorAddrStr] = validatorBondedTokens
			}
			continue
		}

		// If no delegations exist, treat entire validator bonded tokens as validator-only
		// This is the natural default behavior
		if len(delegations) == 0 {
			validatorBondedTokens := validator.GetBondedTokens()
			if !validatorBondedTokens.IsZero() {
				validatorAddrStr := validatorAccAddr.String()
				stakeAmounts[validatorAddrStr] = validatorBondedTokens
			}
			continue
		}

		// Process all delegations (including validator self-delegation)
		// Each delegation represents a distinct stakeholder with their portion of the total stake
		for _, delegation := range delegations {
			delegatorAddr, err := cosmostypes.AccAddressFromBech32(delegation.GetDelegatorAddr())
			if err != nil {
				logger.Error(fmt.Sprintf(
					"failed to parse delegator address %s: %v",
					delegation.GetDelegatorAddr(), err,
				))
				continue
			}

			delegatedShares := delegation.GetShares()
			if !delegatedShares.IsZero() {
				// Convert shares to tokens using the validator's exchange rate
				delegatedTokens := validator.TokensFromShares(delegatedShares).TruncateInt()

				if !delegatedTokens.IsZero() {
					delegatorAddrStr := delegatorAddr.String()
					stakeAmounts[delegatorAddrStr] = delegatedTokens
				}
			}
		}
	}

	if len(stakeAmounts) == 0 {
		logger.Warn("no stakeholders found, skipping reward distribution")
		return nil
	}

	// Calculate proportional rewards using Largest Remainder Method
	rewardAmounts := make(map[string]math.Int)
	totalCalculated := math.ZeroInt()

	// Phase 1: Calculate base rewards and track fractional remainders
	type addressFraction struct {
		address  string
		fraction *big.Rat
	}
	var fractions []addressFraction

	for addrStr, stake := range stakeAmounts {
		// Calculate exact proportional reward using big.Rat
		exactRatio := new(big.Rat).SetFrac(
			new(big.Int).Mul(stake.BigInt(), validatorRewardAmount.BigInt()),
			totalBondedTokens.BigInt(),
		)

		// Split into integer (base reward) and fractional parts
		baseReward := new(big.Int).Quo(exactRatio.Num(), exactRatio.Denom())
		baseRewardInt := math.NewIntFromBigInt(baseReward)
		rewardAmounts[addrStr] = baseRewardInt
		totalCalculated = totalCalculated.Add(baseRewardInt)

		// Calculate fractional remainder
		baseRat := new(big.Rat).SetInt(baseReward)
		fractionalPart := new(big.Rat).Sub(exactRatio, baseRat)

		// Only store non-zero fractions
		if fractionalPart.Sign() > 0 {
			fractions = append(fractions, addressFraction{addrStr, fractionalPart})
		}

		logger.Debug(fmt.Sprintf(
			"  stakeholder %s: stake=%s, base_reward=%s, fraction=%s",
			addrStr,
			stake.String(),
			baseRewardInt.String(),
			fractionalPart.FloatString(6),
		))
	}

	// Phase 2: Distribute remainder tokens using Largest Remainder Method
	remainder := validatorRewardAmount.Sub(totalCalculated)
	tokensToDistribute := remainder.Int64()

	logger.Debug(fmt.Sprintf(
		"Total calculated: %s, target: %s, remainder: %s tokens",
		totalCalculated.String(),
		validatorRewardAmount.String(),
		remainder.String(),
	))

	if tokensToDistribute > 0 && len(fractions) > 0 {
		// Sort by fractional remainder (descending)
		sort.Slice(fractions, func(i, j int) bool {
			return fractions[i].fraction.Cmp(fractions[j].fraction) > 0
		})

		logger.Debug(fmt.Sprintf(
			"Distributing %d remainder tokens using Largest Remainder Method",
			tokensToDistribute,
		))

		// Distribute remainder tokens to addresses with largest fractional remainders
		for t := int64(0); t < tokensToDistribute; t++ {
			addrStr := fractions[t%int64(len(fractions))].address
			rewardAmounts[addrStr] = rewardAmounts[addrStr].AddRaw(1)

			logger.Debug(fmt.Sprintf(
				"  Added 1 token to %s (fraction: %s)",
				addrStr,
				fractions[t%int64(len(fractions))].fraction.FloatString(6),
			))
		}
	} else if tokensToDistribute > 0 {
		// Edge case: remainder exists but no fractional parts
		logger.Warn(fmt.Sprintf(
			"Remainder %d tokens with no fractional parts - adding to first recipient",
			tokensToDistribute,
		))
		// Add to first address in map (deterministic iteration order not guaranteed, but rare edge case)
		for addrStr := range rewardAmounts {
			rewardAmounts[addrStr] = rewardAmounts[addrStr].Add(remainder)
			break
		}
	}

	// Create transfers for all recipients
	totalDistributed := math.ZeroInt()
	for addrStr, rewardAmount := range rewardAmounts {
		totalDistributed = totalDistributed.Add(rewardAmount)

		if rewardAmount.IsZero() {
			logger.Debug(fmt.Sprintf(
				"recipient %s reward is zero, skipping",
				addrStr,
			))
			continue
		}

		// Queue the reward transfer
		rewardCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, rewardAmount)

		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         settlementOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: addrStr,
			Coin:             rewardCoin,
		})

		stake := stakeAmounts[addrStr]
		logger.Info(fmt.Sprintf(
			"queued reward transfer: %s to stakeholder %s (stake: %s, share: %s%%)",
			rewardCoin.String(),
			addrStr,
			stake.String(),
			new(big.Rat).SetFrac(
				stake.BigInt(),
				totalBondedTokens.BigInt(),
			).FloatString(2),
		))
	}

	logger.Info(fmt.Sprintf(
		"validator and delegator reward distribution complete: distributed %s of %s uPOKT to %d recipients",
		totalDistributed.String(),
		validatorRewardAmount.String(),
		len(rewardAmounts),
	))

	return nil
}
