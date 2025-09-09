package token_logic_module

// This file contains the business logic necessary to distribute rewards to validators
// and their delegators.

import (
	"context"
	"fmt"
	"math/big"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/pokt-network/poktroll/app/pocket"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

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
	stakeholderStakeAmounts, sortedStakeAddresses, err := discoverStakeholderStakes(ctx, logger, stakingKeeper, validators)
	if err != nil {
		return err
	}
	if len(stakeholderStakeAmounts) == 0 {
		logger.Warn("SHOULD NEVER HAPPEN: no stakeholders found, skipping reward distribution")
		return nil
	}

	// Step 2: Calculate proportional rewards using Largest Remainder Method
	proportionalRewardAmounts := calculateProportionalRewards(logger, stakeholderStakeAmounts, totalBondedTokens, totalRewardAmount)

	// Step 3: Create and queue reward transfers using sorted addresses for determinism
	return queueRewardTransfers(logger, result, proportionalRewardAmounts, stakeholderStakeAmounts, totalBondedTokens, validators, settlementOpReason, sortedStakeAddresses)
}

// discoverStakeholderStakes discovers all stakeholders and their stakes:
//  1. Identifies all validator delegators
//  2. Collects validator and delegator stake amounts
//  3. Returns both a stake map and deterministically sorted addresses (by stake descending)
//
// DEV_NOTE: The sorted addresses MUST be used when iterating through the map to ensure determinism.
func discoverStakeholderStakes(
	ctx context.Context,
	logger cosmoslog.Logger,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validators []stakingtypes.Validator,
) (map[string]math.Int, []string, error) {
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

	// Sort addresses by stake (descending) to ensure deterministic reward distribution
	sortedAddresses := sortAddressesByStakeDesc(stakeAmounts)
	return stakeAmounts, sortedAddresses, nil
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
	// Step 1: Calculate base proportional rewards
	rewardAmounts := calculateBaseProportionalRewards(logger, stakeAmounts, totalBondedTokens, totalRewardAmount)

	// Step 2: Distribute any remainder using Largest Remainder Method
	applyLargestRemainderMethod(logger, rewardAmounts, stakeAmounts, totalBondedTokens, totalRewardAmount)

	return rewardAmounts
}

// calculateBaseProportionalRewards calculates base integer rewards for each stakeholder.
// Returns reward amounts map with base rewards; fractional parts handled separately by LRM.
func calculateBaseProportionalRewards(
	logger cosmoslog.Logger,
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) map[string]math.Int {
	// A mapping of address -> reward amount for all stakeholders
	rewardAmounts := make(map[string]math.Int)

	for addrStr, stake := range stakeAmounts {
		// Calculate exact proportional reward using big.Rat for precision.
		// Formula: reward = (stake × totalRewardAmount) / totalBondedTokens
		exactReward := new(big.Rat).SetFrac(
			new(big.Int).Mul(stake.BigInt(), totalRewardAmount.BigInt()),
			totalBondedTokens.BigInt(),
		)

		// Extract integer portion as base reward
		baseReward := new(big.Int).Quo(exactReward.Num(), exactReward.Denom())
		baseRewardInt := math.NewIntFromBigInt(baseReward)
		rewardAmounts[addrStr] = baseRewardInt

		// Track fractional remainder for LRM distribution
		baseRat := new(big.Rat).SetInt(baseReward)
		fractionalPart := new(big.Rat).Sub(exactReward, baseRat)

		logger.Debug(fmt.Sprintf(
			"  stakeholder %s: stake=%s, base_reward=%s, fraction=%s",
			addrStr,
			stake.String(),
			baseRewardInt.String(),
			fractionalPart.FloatString(6),
		))
	}

	return rewardAmounts
}

// applyLargestRemainderMethod distributes remainder tokens using LRM.
// Ensures exact token distribution by allocating remainders to addresses
// with the largest fractional parts first.
// DEV_NOTE: This function transforms the rewardAmounts map in place.
func applyLargestRemainderMethod(
	logger cosmoslog.Logger,
	rewardAmounts map[string]math.Int,
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
) {
	// Compute the total distributed reward amount
	totalDistributedRewardAmount := math.ZeroInt()
	for _, amount := range rewardAmounts {
		totalDistributedRewardAmount = totalDistributedRewardAmount.Add(amount)
	}

	// Calculate remainder tokens to distribute
	remainder := totalRewardAmount.Sub(totalDistributedRewardAmount).Int64()

	logger.Debug(fmt.Sprintf(
		"Applying the largest remainder method to reward distribution. Total amount distributed: %s. Total reward amount: %s. Remainder: %d tokens",
		totalDistributedRewardAmount.String(),
		totalRewardAmount.String(),
		remainder,
	))

	if remainder > 0 {
		distributeRemainderTokens(logger, rewardAmounts, stakeAmounts, totalBondedTokens, totalRewardAmount, remainder)
	}
}

// distributeRemainderTokens allocates remainder tokens using LRM.
// Distributes to addresses with largest fractional remainders first.
func distributeRemainderTokens(
	logger cosmoslog.Logger,
	rewardAmounts map[string]math.Int,
	stakeAmounts map[string]math.Int,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
	tokensToDistribute int64,
) {
	// Sort addresses by fractional remainder (descending) for LRM distribution
	addressesByFractionDesc := sortAddressesByFractionDesc(stakeAmounts, totalBondedTokens, totalRewardAmount)
	numAddresses := int64(len(addressesByFractionDesc))

	// Sanity check: remainder should only exist if addresses have fractional parts
	if numAddresses == 0 {
		logger.Error(fmt.Sprintf(
			"SHOULD NEVER HAPPEN: remainder %d tokens to distribute but no addresses with fractional parts found. This indicates a bug in the reward calculation logic.",
			tokensToDistribute,
		))
		// TODO_INVESTIGATE: This edge case should be investigated further to understand when it might occur
		return
	}

	logger.Debug(fmt.Sprintf(
		"Distributing %d remainder tokens to %d addresses with fractional parts (by LRM ordering)",
		tokensToDistribute,
		numAddresses,
	))

	// Each address gets either:
	//   - baseTokensPerAddr + 1 (first 'extraTokensNeeded' addresses)
	//   - baseTokensPerAddr (remaining addresses)
	baseTokensPerAddr := tokensToDistribute / numAddresses
	extraTokensNeeded := tokensToDistribute % numAddresses

	for i, addrStr := range addressesByFractionDesc {
		tokensForThisAddr := baseTokensPerAddr
		// Distribute extra tokens to addresses with largest fractions
		if int64(i) < extraTokensNeeded {
			tokensForThisAddr++
		}

		if tokensForThisAddr > 0 {
			rewardAmounts[addrStr] = rewardAmounts[addrStr].AddRaw(tokensForThisAddr)
			logger.Debug(fmt.Sprintf(
				"  Added %d tokens to %s (by stake order)",
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
	sortedStakeAddresses []string,
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

	// Queue ModToAcctTransfer for each recipient in deterministic order
	for _, addrStr := range sortedStakeAddresses {
		rewardAmount := rewardAmounts[addrStr]
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
