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

// addressWithFraction stores addresses with their fractional parts; used for sorting addresses by fraction.
type addressWithFraction struct {
	address  string
	fraction *big.Rat
}

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
	totalRewardCoin cosmostypes.Coin,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeValidatorRewards",
		"session_id", result.GetSessionId(),
		"total_reward_amount", totalRewardCoin.Amount,
	)

	// Step 1: Validate inputs and prepare validator data
	validators, totalBondedTokens, err := validateAndPrepareValidatorRewards(ctx, logger, stakingKeeper, totalRewardCoin.Amount)
	if err != nil {
		return err
	}
	if validators == nil {
		// Skip distribution (zero amount, no validators, etc.)
		return nil
	}

	// Step 2: Distribute rewards to validators and their delegators
	return distributeToValidatorsAndDelegators(
		ctx,
		logger,
		result,
		stakingKeeper,
		validators,
		totalBondedTokens,
		totalRewardCoin.Amount,
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

// distributeToValidatorsAndDelegators distributes rewards to validators and their delegators
// based purely on stake proportions without any commission calculations.
func distributeToValidatorsAndDelegators(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validators []stakingtypes.Validator,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeToValidatorsAndDelegators",
		"session_id", result.GetSessionId(),
		"total_reward_amount", totalRewardAmount,
	)

	logger.Info(fmt.Sprintf(
		"distributing %s to validators and delegators based on stake weight (total bonded: %s)",
		totalRewardAmount.String(),
		totalBondedTokens.String(),
	))

	// Step 1: Discover all stakeholders and their stakes
	stakeAmounts, err := discoverStakeholderStakes(ctx, logger, stakingKeeper, validators)
	if err != nil {
		return err
	}
	if len(stakeAmounts) == 0 {
		logger.Warn("no stakeholders found, skipping reward distribution")
		return nil
	}

	// Step 2: Calculate proportional rewards using Largest Remainder Method
	rewardAmounts := calculateProportionalRewards(logger, stakeAmounts, totalBondedTokens, totalRewardAmount)

	// Step 3: Create and queue reward transfers
	return queueRewardTransfers(logger, result, rewardAmounts, stakeAmounts, totalBondedTokens, settlementOpReason)
}

// discoverStakeholderStakes discovers all validators and delegators and collects their stake amounts.
// Returns a map of address -> stake amount for all stakeholders.
func discoverStakeholderStakes(
	ctx context.Context,
	logger cosmoslog.Logger,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validators []stakingtypes.Validator,
) (map[string]math.Int, error) {
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
			// On delegation query error, treat the entire validator bonded tokens as self-bonded rewards
			// This maintains backward compatibility and ensures validators still receive rewards despite delegation query failures
			logger.Error(fmt.Sprintf(
				"failed to get delegations for validator %s: %v (using no delegators distribution)",
				validator.GetOperator(), err,
			))

			validatorBondedTokens := validator.GetBondedTokens()
			if validatorBondedTokens.IsZero() {
				continue
			}

			validatorAddrStr := validatorAccAddr.String()
			stakeAmounts[validatorAddrStr] = validatorBondedTokens
		}

		// If no delegations exist, all bonded tokens are validator's stake
		if len(delegations) == 0 {
			validatorBondedTokens := validator.GetBondedTokens()
			if validatorBondedTokens.IsZero() {
				continue
			}

			validatorAddrStr := validatorAccAddr.String()
			stakeAmounts[validatorAddrStr] = validatorBondedTokens
		}

		// Extract and record stake amounts from delegations
		collectDelegationStakes(logger, validator, delegations, stakeAmounts)
	}

	return stakeAmounts, nil
}

// collectDelegationStakes extracts and records stake amounts from a validator's delegations.
// Converts each delegation's shares to tokens and records them individually in stakeAmounts.
func collectDelegationStakes(
	logger cosmoslog.Logger,
	validator stakingtypes.Validator,
	delegations []stakingtypes.Delegation,
	stakeAmounts map[string]math.Int,
) {
	// Extract and record stake amounts for each delegator (including validator self-delegation)
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

// calculateProportionalRewards calculates rewards for each stakeholder using the Largest Remainder Method.
// This ensures precise and fair proportional distribution with no remainder left unallocated.
func calculateProportionalRewards(
	logger cosmoslog.Logger,
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) map[string]math.Int {
	// Step 1: Calculate base proportional rewards and collect addresses with fractional remainders
	rewardAmounts, addressesByFraction := calculateBaseProportionalRewards(logger, stakeAmounts, totalBondedTokens, totalRewardAmount)

	// Step 2: Distribute any remainder using Largest Remainder Method
	applyLargestRemainderMethod(logger, rewardAmounts, addressesByFraction, totalRewardAmount)

	return rewardAmounts
}

// calculateBaseProportionalRewards calculates base integer rewards for each stakeholder
// and returns both the reward amounts and addresses sorted by their fractional remainders.
func calculateBaseProportionalRewards(
	logger cosmoslog.Logger,
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) (map[string]math.Int, []string) {
	rewardAmounts := make(map[string]math.Int)

	var addressesToSort []addressWithFraction

	for addrStr, stake := range stakeAmounts {
		// Calculate exact proportional reward using big.Rat for maximum precision
		// Formula: stakeholderReward = totalRewardAmount × (stake / totalBondedTokens)
		// Rewritten as: stakeholderReward = (stake × totalRewardAmount) / totalBondedTokens
		// This order prevents precision loss from calculating small fractions first
		exactReward := new(big.Rat).SetFrac(
			new(big.Int).Mul(stake.BigInt(), totalRewardAmount.BigInt()),
			totalBondedTokens.BigInt(),
		)

		// Split into integer (base reward) and fractional parts
		baseReward := new(big.Int).Quo(exactReward.Num(), exactReward.Denom())
		baseRewardInt := math.NewIntFromBigInt(baseReward)
		rewardAmounts[addrStr] = baseRewardInt

		// Calculate fractional remainder
		baseRat := new(big.Rat).SetInt(baseReward)
		fractionalPart := new(big.Rat).Sub(exactReward, baseRat)

		// Only collect addresses with non-zero fractions for sorting
		if fractionalPart.Sign() > 0 {
			addressesToSort = append(addressesToSort, addressWithFraction{addrStr, fractionalPart})
		}

		logger.Debug(fmt.Sprintf(
			"  stakeholder %s: stake=%s, base_reward=%s, fraction=%s",
			addrStr,
			stake.String(),
			baseRewardInt.String(),
			fractionalPart.FloatString(6),
		))
	}

	// Sort addresses by their fractional remainders (descending)
	sort.Slice(addressesToSort, func(i, j int) bool {
		return addressesToSort[i].fraction.Cmp(addressesToSort[j].fraction) > 0
	})

	// Extract just the sorted addresses
	addressesByFraction := make([]string, len(addressesToSort))
	for i, item := range addressesToSort {
		addressesByFraction[i] = item.address
	}

	return rewardAmounts, addressesByFraction
}

// applyLargestRemainderMethod distributes remainder tokens to addresses with the largest fractional parts.
// This ensures all tokens are distributed while maintaining proportional fairness.
func applyLargestRemainderMethod(
	logger cosmoslog.Logger,
	rewardAmounts map[string]math.Int,
	addressesByFraction []string,
	totalRewardAmount math.Int,
) {
	totalDistributedRewardAmount := math.ZeroInt()
	for _, amount := range rewardAmounts {
		totalDistributedRewardAmount = totalDistributedRewardAmount.Add(amount)
	}

	remainder := totalRewardAmount.Sub(totalDistributedRewardAmount).Int64()

	logger.Debug(fmt.Sprintf(
		"Total calculated: %s, target: %s, remainder: %d tokens",
		totalDistributedRewardAmount.String(),
		totalRewardAmount.String(),
		remainder,
	))

	if remainder > 0 {
		distributeRemainderTokens(logger, rewardAmounts, addressesByFraction, remainder)
	}
}

// distributeRemainderTokens allocates remainder tokens to addresses with the largest fractional remainders.
func distributeRemainderTokens(
	logger cosmoslog.Logger,
	rewardAmounts map[string]math.Int,
	addressesByFraction []string,
	tokensToDistribute int64,
) {
	if len(addressesByFraction) == 0 {
		// Edge case: remainder exists but no fractional parts
		logger.Warn(fmt.Sprintf(
			"Remainder %d tokens with no fractional parts - adding to first recipient",
			tokensToDistribute,
		))
		// Add to first address in map (deterministic iteration order not guaranteed, but rare edge case)
		for addrStr := range rewardAmounts {
			remainder := math.NewInt(tokensToDistribute)
			rewardAmounts[addrStr] = rewardAmounts[addrStr].Add(remainder)
			break
		}
		return
	}

	// Addresses are already sorted by fractional remainder (descending) from calculateBaseProportionalRewards
	logger.Debug(fmt.Sprintf(
		"Distributing %d remainder tokens using Largest Remainder Method",
		tokensToDistribute,
	))

	// Calculate how many tokens each address gets: some get (tokensToDistribute / numAddresses) + 1,
	// others get (tokensToDistribute / numAddresses)
	numAddresses := int64(len(addressesByFraction))
	baseTokensPerAddr := tokensToDistribute / numAddresses
	extraTokensNeeded := tokensToDistribute % numAddresses

	for i, addrStr := range addressesByFraction {
		tokensForThisAddr := baseTokensPerAddr
		// First 'extraTokensNeeded' addresses get one extra token
		if int64(i) < extraTokensNeeded {
			tokensForThisAddr++
		}

		if tokensForThisAddr > 0 {
			rewardAmounts[addrStr] = rewardAmounts[addrStr].AddRaw(tokensForThisAddr)

			logger.Debug(fmt.Sprintf(
				"  Added %d tokens to %s (largest remaining fraction)",
				tokensForThisAddr,
				addrStr,
			))
		}
	}
}

// queueRewardTransfers creates and queues reward transfers for all recipients.
func queueRewardTransfers(
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	rewardAmounts map[string]math.Int,
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
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
		"validator and delegator reward distribution complete: distributed %s to %d recipients",
		totalDistributed.String(),
		len(rewardAmounts),
	))

	return nil
}
