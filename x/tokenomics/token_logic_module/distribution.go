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

// addressWithFraction stores addresses with their fractional reward parts.
// Used for sorting addresses by fraction.
type addressWithFraction struct {
	address  string
	fraction *big.Rat
}

// sortAddressesByFracDesc sorts addresses by their fractional remainders (descending).
func sortAddressesByFracDesc(addressesToSort []addressWithFraction) []string {
	// Sort addresses by their fractional remainders (descending)
	sort.Slice(addressesToSort, func(i, j int) bool {
		return addressesToSort[i].fraction.Cmp(addressesToSort[j].fraction) > 0
	})

	// Extract just the sorted addresses
	sortedAddressesByFraction := make([]string, len(addressesToSort))
	for i, item := range addressesToSort {
		sortedAddressesByFraction[i] = item.address
	}

	return sortedAddressesByFraction
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

// distributeValidatorRewards distributes session settlement rewards to
// all bonded validators and their delegators.
//
// Specifically:
//   - Validator stake weight is used to distribute rewards regardless of who the block proposer is.
//   - Commission is not taken into account since this is independent of consensus rewards.
//   - The validator's self-bonded and delegated stake is taken into account.
//   - Delegators receive rewards proportional to their delegated stake.
//
// For a stakeholder (self-bonded validator or delegator), the distribution formula is:
//
//	stakeholderReward = totalValidatorRewardAmount × (stakeholderStake / totalBondedStake)
func distributeValidatorRewards(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	totalValidatorRewardCoin cosmostypes.Coin,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeValidatorRewards",
		"session_id", result.GetSessionId(),
		"total_reward_amount", totalValidatorRewardCoin.Amount,
	)

	// Step 1: Validate inputs and prepare validator data
	validators, totalValidatorBondedTokens, err := validateAndPrepareValidatorRewards(ctx, logger, stakingKeeper, totalValidatorRewardCoin.Amount)
	if err != nil {
		return err
	}
	if validators == nil {
		logger.Warn("SHOULD NEVER HAPPEN: Validator set is empty. Skipping validator reward distribution altogether.")
		return nil
	}

	// Step 2: Distribute rewards to validators and their delegators
	return distributeRewardsToValidatorsAndDelegators(
		ctx,
		logger,
		result,
		stakingKeeper,
		validators,
		totalValidatorBondedTokens,
		totalValidatorRewardCoin.Amount,
		settlementOpReason,
	)
}

// validateAndPrepareValidatorRewards prepares validator data for distribution.
// Returns one of:
//  1. (validators, totalValidatorBondedTokens) if distribution should proceed; includes self-bonded and delegated tokens.
//  2. (nil, zero) if distribution should be skipped.
func validateAndPrepareValidatorRewards(
	ctx context.Context,
	logger cosmoslog.Logger,
	stakingKeeper tokenomicstypes.StakingKeeper,
	totalValidatorRewardAmount math.Int,
) ([]stakingtypes.Validator, math.Int, error) {
	if totalValidatorRewardAmount.IsZero() {
		logger.Debug("SHOULD NEVER HAPPEN: validator reward amount is zero, skipping distribution")
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
		logger.Warn("SHOULD NEVER HAPPEN: no bonded validators found, skipping validator reward distribution")
		return nil, math.ZeroInt(), nil
	}

	// Calculate total bonded tokens across all validators
	totalValidatorBondedTokens := math.ZeroInt()
	for _, validator := range validators {
		totalValidatorBondedTokens = totalValidatorBondedTokens.Add(validator.GetBondedTokens())
	}
	if totalValidatorBondedTokens.IsZero() {
		logger.Warn("SHOULD NEVER HAPPEN: total bonded tokens is zero, skipping validator reward distribution")
		return nil, math.ZeroInt(), nil
	}

	logger.Info(fmt.Sprintf(
		"distributing %s to %d validators based on stake weight (total bonded: %s)",
		totalValidatorRewardAmount.String(),
		len(validators),
		totalValidatorBondedTokens.String(),
	))

	return validators, totalValidatorBondedTokens, nil
}

// distributeRewardsToValidatorsAndDelegators distributes rewards to validators and their delegators.
// Rewards are distributed based purely on stake proportions without any commission calculations.
//
// The implementation is composed of three main steps:
//
//  1. Discover all stakeholders and their stakes
//  2. Calculate proportional rewards using Largest Remainder Method
//  3. Create and queue reward transfers
func distributeRewardsToValidatorsAndDelegators(
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
	stakeholderStakeAmounts, err := discoverStakeholderStakes(ctx, logger, stakingKeeper, validators)
	if err != nil {
		return err
	}
	if len(stakeholderStakeAmounts) == 0 {
		logger.Warn("SHOULD NEVER HAPPEN: no stakeholders found, skipping reward distribution")
		return nil
	}

	// Step 2: Calculate proportional rewards using Largest Remainder Method
	proportionalRewardAmounts := calculateProportionalRewards(logger, stakeholderStakeAmounts, totalBondedTokens, totalRewardAmount)

	// Step 3: Create and queue reward transfers
	return queueRewardTransfers(logger, result, proportionalRewardAmounts, stakeholderStakeAmounts, totalBondedTokens, validators, settlementOpReason)
}

// discoverStakeholderStakes does the following:
//  1. Discovers all validator delegators
//  2. Collects the validator and delegator stake amounts
//  3. Returns a map of address -> stake amount for all stakeholders (validators + delegators)
func discoverStakeholderStakes(
	ctx context.Context,
	logger cosmoslog.Logger,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validators []stakingtypes.Validator,
) (map[string]math.Int, error) {
	// A mapping of address -> stake amount for all stakeholders (validators + delegators)
	stakeAmounts := make(map[string]math.Int)

	// Process each validator and their delegators
	for _, validator := range validators {
		valAddr, err := cosmostypes.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			logger.Error(fmt.Sprintf(
				"Failed to parse validator operator address %s: %v. Skipping to the next one.",
				validator.GetOperator(), err,
			))
			continue
		}
		validatorAccAddr := cosmostypes.AccAddress(valAddr)

		validatorBondedTokens := validator.GetBondedTokens()
		if validatorBondedTokens.IsZero() {
			logger.Warn(fmt.Sprintf(
				"SHOULD NEVER HAPPEN: Validator %s has zero bonded tokens. Skipping to the next one.",
				validator.GetOperator(),
			))
			continue
		}

		// Retrieve all delegations for the validator
		delegations, err := stakingKeeper.GetValidatorDelegations(ctx, valAddr)
		if err != nil {

			validatorAddrStr := validatorAccAddr.String()
			stakeAmounts[validatorAddrStr] = validatorBondedTokens

			// On delegation query error, treat the entire validator bonded tokens as self-bonded rewards
			// This maintains backward compatibility and ensures validators still receive rewards despite delegation query failures
			logger.Warn(fmt.Sprintf(
				"SHOULD NEVER HAPPEN: Failed to get delegations for validator %s: %v. Using no delegators distribution. Validator will receive rewards based on all of its bonded tokens.",
				validator.GetOperator(), err,
			))

			continue
		}

		// If no delegations exist, all bonded tokens are validator's stake
		if len(delegations) == 0 {
			validatorAddrStr := validatorAccAddr.String()
			stakeAmounts[validatorAddrStr] = validatorBondedTokens

			logger.Debug(fmt.Sprintf(
				"Validator %s has no delegations. Using no delegators distribution. Validator will receive rewards based on all of its bonded tokens.",
				validator.GetOperator(),
			))
			continue
		}

		// Extract and record stake amounts from delegations
		// DEV_NOTE: This transforms the stakeAmounts map in place.
		collectDelegationStakes(logger, validator, delegations, stakeAmounts)
	}

	return stakeAmounts, nil
}

// collectDelegationStakes extracts and records stake amounts from a validator's delegations.
// Converts each delegation's shares to tokens and records them individually in stakeAmounts.
// DEV_NOTE: This transforms the stakeAmounts map in place.
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
			logger.Error(fmt.Sprintf("SHOULD NEVER HAPPEN: failed to parse delegator address %s: %v. Skipping to the next one...", delegation.GetDelegatorAddr(), err))
			continue
		}
		delegatorAddrStr := delegatorAddr.String()

		delegatedShares := delegation.GetShares()
		if !delegatedShares.IsZero() {
			// Convert shares to tokens using the validator's exchange rate
			delegatedTokens := validator.TokensFromShares(delegatedShares).TruncateInt()
			if delegatedTokens.IsZero() {
				logger.Warn(fmt.Sprintf("SHOULD NEVER HAPPEN: delegator %s has zero delegated tokens but the delegated share exists. Skipping to the next one...", delegatorAddrStr))
				continue
			}
			stakeAmounts[delegatorAddrStr] = delegatedTokens
		}
	}
}

// calculateProportionalRewards calculates rewards for each stakeholder.
// It uses the "Largest Remainder Method" to ensure precise and fair proportional
// distribution with no remainder left unallocated.
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

// calculateBaseProportionalRewards calculates base integer rewards for each stakeholder.
// It returns both the reward amounts and addresses sorted by their fractional remainders.
func calculateBaseProportionalRewards(
	logger cosmoslog.Logger,
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) (map[string]math.Int, []string) {
	// A mapping of address -> reward amount for all stakeholders
	rewardAmounts := make(map[string]math.Int)

	// Addresses receiving rewards.
	// Needed to sort by their fractional remainders
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

	// Sort addresses by their fractional remainders in descending order
	sortedAddressesByFraction := sortAddressesByFracDesc(addressesToSort)

	// Return the reward amounts and sorted addresses by their fractional remainders
	return rewardAmounts, sortedAddressesByFraction
}

// applyLargestRemainderMethod distributes remainder tokens.
// It allocates remainder tokens to addresses with the largest fractional parts.
// This ensures all tokens are distributed while maintaining proportional fairness.
// DEV_NOTE: This function transforms the rewardAmounts map in place.
func applyLargestRemainderMethod(
	logger cosmoslog.Logger,
	rewardAmounts map[string]math.Int,
	addressesByFraction []string,
	totalRewardAmount math.Int,
) {
	// Compute the total distributed reward amount
	totalDistributedRewardAmount := math.ZeroInt()
	for _, amount := range rewardAmounts {
		totalDistributedRewardAmount = totalDistributedRewardAmount.Add(amount)
	}

	// Compute the remainder by comparing total reward and total distributed reward amount
	remainder := totalRewardAmount.Sub(totalDistributedRewardAmount).Int64()

	logger.Debug(fmt.Sprintf(
		"Applying the largest remainder method to reward distribution. Total amount distributed: %s. Total reward amount: %s. Remainder: %d tokens",
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
	// Edge case: remainder exists but no fractional parts
	if len(addressesByFraction) == 0 {
		// Add to first address in map (deterministic iteration order not guaranteed, but rare edge case)
		for addrStr := range rewardAmounts {
			logger.Warn(fmt.Sprintf(
				"Remainder %d tokens but no fractional parts found. Adding to first recipient so tokens are not lost: %s.",
				tokensToDistribute,
				addrStr,
			))

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

	// Calculate how many tokens each address gets. Will be one of:
	// 	(tokensToDistribute / numAddresses) + 1
	// 	(tokensToDistribute / numAddresses)
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
	validators []stakingtypes.Validator,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With("method", "queueRewardTransfers")

	// Use for logging purposes only
	totalDistributed := math.ZeroInt()

	// Build a set of validator addresses to easily retrieve their delegators
	// from the stakeAmounts map.
	validatorAddresses := make(map[string]bool)
	for _, validator := range validators {
		valAddr, err := cosmostypes.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			continue
		}
		validatorAccAddr := cosmostypes.AccAddress(valAddr)
		validatorAddresses[validatorAccAddr.String()] = true
	}

	// Queue a ModToAcctTransfer for each recipient with the appropriate operation reason.
	for addrStr, rewardAmount := range rewardAmounts {
		if rewardAmount.IsZero() {
			logger.Debug(fmt.Sprintf(
				"SHOULD RARELY HAPPEN: recipient %s reward is zero, skipping",
				addrStr,
			))
			continue
		}

		// Account for the total distributed amount
		totalDistributed = totalDistributed.Add(rewardAmount)
		rewardCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, rewardAmount)

		// Determine if this is a delegator or validator reward
		isValidator := validatorAddresses[addrStr]
		actualRewardOpReason := settlementOpReason
		recipientType := "validator"

		// This is a delegator reward - use delegator operation reason
		if !isValidator {
			recipientType = "delegator"

			// Update the op reason ac
			switch settlementOpReason {

			// Mint = Burn
			case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:
				actualRewardOpReason = tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION

			// TLM Global Mint
			case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION:
				actualRewardOpReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION
			}
		}

		// Queue the reward transfer with the appropriate operation reason
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         actualRewardOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: addrStr,
			Coin:             rewardCoin,
		})

		stake := stakeAmounts[addrStr]
		logger.Info(fmt.Sprintf(
			"queued reward transfer: %s to %s %s (stake: %s, share: %s%%)",
			rewardCoin.String(),
			recipientType,
			addrStr,
			stake.String(),
			new(big.Rat).SetFrac(
				stake.BigInt(),
				totalBondedTokens.BigInt(),
			).FloatString(2),
		))
	}

	logger.Info(fmt.Sprintf(
		"validator and delegator reward distribution complete: distributed %s to %d validators and %d total stakeholders",
		totalDistributed.String(),
		len(validatorAddresses),
		len(rewardAmounts),
	))

	return nil
}
