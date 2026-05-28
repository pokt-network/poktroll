package token_logic_module

// This file contains the business logic necessary to distribute rewards to validators
// and their delegators.

import (
	"context"
	"fmt"
	"sort"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/pokt-network/poktroll/app/pocket"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// DistributeValidatorRewards distributes session settlement rewards to
// all bonded validators and their delegators.
//
// Specifically:
//   - The total reward is first split across all bonded validators proportional
//     to each validator's total bonded stake (self-bonded + delegated).
//   - Each validator's commission rate (from the staking module) is applied to its
//     pool share: the commission is paid directly to the validator operator account,
//     mirroring the Cosmos consensus-reward model.
//   - The post-commission remainder is distributed to that validator's stakeholders
//     (including the validator's self-delegation) proportional to their delegated stake.
//
// The distribution is computed in two levels, each closed exactly via the Largest
// Remainder Method (LRM) so that no upokt is left unallocated:
//
//	Level 1 (across validators):
//	  poolShare_v   = totalReward × (validatorBondedTokens_v / totalBondedTokens)
//
//	Level 2 (within each validator):
//	  commission_v  = floor(poolShare_v × commissionRate_v)        → validator account
//	  remainder_v   = poolShare_v − commission_v
//	  delegatorReward = remainder_v × (delegatorStake / validatorTotalDelegatedStake)
func DistributeValidatorRewards(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	totalValidatorRewardCoin cosmostypes.Coin,
	settlementOpReason tokenomicstypes.SettlementOpReason,
	sessionEndHeight int64,
) error {
	logger = logger.With(
		"method", "DistributeValidatorRewards",
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

	// Step 2: Distribute rewards to validators (with commission) and their delegators
	return distributeRewardsToValidatorsAndDelegators(
		ctx,
		logger,
		result,
		stakingKeeper,
		validators,
		totalValidatorBondedTokens,
		totalValidatorRewardCoin.Amount,
		settlementOpReason,
		sessionEndHeight,
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

// validatorPoolEntry holds a single validator and the account address derived from
// its operator address, in the order rewards are distributed.
type validatorPoolEntry struct {
	validator stakingtypes.Validator
	accAddr   string
}

// distributeRewardsToValidatorsAndDelegators distributes rewards to validators and
// their delegators using a two-level, commission-aware allocation.
//
// The implementation is composed of three main steps:
//
//  1. Split the total reward across validators by bonded-stake weight (Level 1 LRM).
//  2. For each validator, carve out its commission and split the remainder across
//     its delegators by delegated-stake weight (Level 2 LRM).
//  3. Queue one reward transfer per recipient in deterministic order.
func distributeRewardsToValidatorsAndDelegators(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validators []stakingtypes.Validator,
	totalBondedTokens math.Int,
	totalRewardAmount math.Int,
	settlementOpReason tokenomicstypes.SettlementOpReason,
	sessionEndHeight int64,
) error {
	logger = logger.With(
		"method", "distributeToValidatorsAndDelegators",
		"session_id", result.GetSessionId(),
		"total_reward_amount", totalRewardAmount,
	)

	// Step 1: Build the per-validator stake map and split the total reward across
	// validators proportional to their bonded stake (Level 1 LRM).
	validatorEntries, validatorStakeAmounts, totalValidatorStake := buildValidatorStakes(logger, validators)
	if len(validatorEntries) == 0 {
		logger.Warn("SHOULD NEVER HAPPEN: no eligible validators found, skipping reward distribution")
		return nil
	}

	validatorPoolShares := calculateProportionalRewards(logger, validatorStakeAmounts, totalValidatorStake, totalRewardAmount)

	// Log both the validator-set total (the chain-level number passed in from
	// validateAndPrepareValidatorRewards) AND the eligible-stake total (the
	// actual denominator used for the proportional split, which excludes any
	// validators dropped by buildValidatorStakes — e.g., unparseable operator
	// addresses). The two will normally match; surfacing both makes drift
	// observable in audit logs.
	logger.Info(fmt.Sprintf(
		"distributing %s across %d validators by stake weight (eligible_stake: %s, validator_set_total: %s)",
		totalRewardAmount.String(),
		len(validatorEntries),
		totalValidatorStake.String(),
		totalBondedTokens.String(),
	))

	// rewardAmounts accumulates the final reward per recipient account address.
	// A recipient may receive from multiple sources (e.g. a validator's commission
	// plus its self-delegation slice, or a delegator delegating to several validators).
	rewardAmounts := make(map[string]math.Int)

	// validatorAccAddresses identifies which recipients are validator operator accounts,
	// used only to pick the op-reason label when queueing transfers.
	validatorAccAddresses := make(map[string]bool, len(validatorEntries))
	for _, entry := range validatorEntries {
		validatorAccAddresses[entry.accAddr] = true
	}

	// Step 2: For each validator (deterministic order), apply commission and distribute
	// the remainder to its delegators (Level 2 LRM).
	for _, entry := range validatorEntries {
		poolShare := validatorPoolShares[entry.accAddr]
		if poolShare.IsZero() {
			continue
		}

		// Carve out the validator's commission and pay it directly to the operator account.
		commissionRate := entry.validator.Commission.Rate
		commission := calculateCommission(poolShare, commissionRate)
		if commission.IsPositive() {
			addReward(rewardAmounts, entry.accAddr, commission)
			logger.Debug(fmt.Sprintf(
				"validator %s commission: %s (rate %s of pool share %s)",
				entry.accAddr, commission.String(), commissionRate.String(), poolShare.String(),
			))
		}

		remainder := poolShare.Sub(commission)
		if !remainder.IsPositive() {
			// All of the pool share was taken as commission (100% commission rate).
			if err := emitValidatorRewardEvent(ctx, validatorRewardSummary{
				sessionEndHeight:     sessionEndHeight,
				opReason:             settlementOpReason,
				entry:                entry,
				commissionRate:       commissionRate,
				poolShare:            poolShare,
				commission:           commission,
				selfDelegationReward: math.ZeroInt(),
				delegatorsReward:     math.ZeroInt(),
				totalDelegatedStake:  math.ZeroInt(),
				numDelegators:        0,
			}); err != nil {
				return err
			}
			continue
		}

		// Discover this validator's delegators (including its self-delegation).
		delegatorStakeAmounts, totalDelegatedStake, ok := collectValidatorDelegationStakes(ctx, logger, stakingKeeper, entry.validator, entry.accAddr)

		// On query failure or absence of delegations, the validator is the sole
		// stakeholder: the remainder is paid to the operator account as well.
		// We log distinctly here for operator diagnostics — the three trigger
		// conditions look identical from this branch but have very different
		// causes:
		//
		//   - !ok                                — staking-keeper query failed
		//     (already logged Warn inside collectValidatorDelegationStakes)
		//   - len(delegatorStakeAmounts) == 0    — no delegations on a BONDED
		//     validator. This SHOULD be unreachable since the staking module
		//     enforces a self-delegation at bond time, but a genesis-imported
		//     validator that bypassed MsgDelegate could trip this. We Warn
		//     because the remainder going entirely to the operator is the
		//     correct behavior here, but the operator-side missing-data is
		//     itself a fact worth surfacing.
		//   - totalDelegatedStake.IsZero()       — all delegations had zero
		//     shares. Also surprising for a bonded validator.
		//
		// All three paths credit the full remainder to the operator account;
		// this matches the previous behavior. Only the visibility changes.
		if !ok || len(delegatorStakeAmounts) == 0 || totalDelegatedStake.IsZero() {
			switch {
			case !ok:
				// already-logged by callee; nothing to add here.
			case len(delegatorStakeAmounts) == 0 && entry.validator.GetBondedTokens().IsPositive():
				logger.Warn(fmt.Sprintf(
					"Validator %s is bonded with %s tokens but has zero delegations. Crediting full remainder %s to operator %s.",
					entry.validator.GetOperator(),
					entry.validator.GetBondedTokens().String(),
					remainder.String(),
					entry.accAddr,
				))
			case totalDelegatedStake.IsZero():
				logger.Warn(fmt.Sprintf(
					"Validator %s has delegations but total delegated stake is zero (all shares zero). Crediting full remainder %s to operator %s.",
					entry.validator.GetOperator(), remainder.String(), entry.accAddr,
				))
			}
			addReward(rewardAmounts, entry.accAddr, remainder)
			if err := emitValidatorRewardEvent(ctx, validatorRewardSummary{
				sessionEndHeight:     sessionEndHeight,
				opReason:             settlementOpReason,
				entry:                entry,
				commissionRate:       commissionRate,
				poolShare:            poolShare,
				commission:           commission,
				selfDelegationReward: remainder,
				delegatorsReward:     math.ZeroInt(),
				totalDelegatedStake:  math.ZeroInt(),
				numDelegators:        0,
			}); err != nil {
				return err
			}
			continue
		}

		// Distribute the remainder across the validator's delegators by stake weight (Level 2 LRM).
		delegatorRewards := calculateProportionalRewards(logger, delegatorStakeAmounts, totalDelegatedStake, remainder)

		// Split the remainder into the validator's self-delegation slice and the
		// external delegators' total for the per-validator summary event.
		//
		// The operations below (accumulation, single-key match, count) are individually
		// order-independent, but we iterate in sorted-address order anyway: it is
		// defense-in-depth on a consensus path, so the loop stays deterministic by
		// construction even if a future edit adds an order-dependent side effect.
		selfDelegationReward := math.ZeroInt()
		numExternalDelegators := uint32(0)
		sortedDelegators := make([]string, 0, len(delegatorRewards))
		for delAddr := range delegatorRewards {
			sortedDelegators = append(sortedDelegators, delAddr)
		}
		sort.Strings(sortedDelegators)
		for _, delAddr := range sortedDelegators {
			delReward := delegatorRewards[delAddr]
			addReward(rewardAmounts, delAddr, delReward)
			if delAddr == entry.accAddr {
				selfDelegationReward = delReward
			} else {
				numExternalDelegators++
			}
		}
		delegatorsReward := remainder.Sub(selfDelegationReward)

		if err := emitValidatorRewardEvent(ctx, validatorRewardSummary{
			sessionEndHeight:     sessionEndHeight,
			opReason:             settlementOpReason,
			entry:                entry,
			commissionRate:       commissionRate,
			poolShare:            poolShare,
			commission:           commission,
			selfDelegationReward: selfDelegationReward,
			delegatorsReward:     delegatorsReward,
			totalDelegatedStake:  totalDelegatedStake,
			numDelegators:        numExternalDelegators,
		}); err != nil {
			return err
		}
	}

	// Step 3: Queue one transfer per recipient in deterministic order.
	return queueRewardTransfers(logger, result, rewardAmounts, validatorAccAddresses, settlementOpReason, totalRewardAmount)
}

// validatorRewardSummary holds the per-validator reward breakdown emitted as an
// EventValidatorRewardDistribution.
type validatorRewardSummary struct {
	sessionEndHeight     int64
	opReason             tokenomicstypes.SettlementOpReason
	entry                validatorPoolEntry
	commissionRate       math.LegacyDec
	poolShare            math.Int
	commission           math.Int
	selfDelegationReward math.Int
	delegatorsReward     math.Int
	totalDelegatedStake  math.Int
	numDelegators        uint32
}

// emitValidatorRewardEvent emits a single EventValidatorRewardDistribution summarizing how a
// validator's pool share was split into commission and (self/external) delegator rewards.
// It is emitted once per validator per op_reason per settlement block — bounded by the
// validator-set size, so it does NOT scale with delegators or claims (#1758 preserved).
func emitValidatorRewardEvent(ctx context.Context, summary validatorRewardSummary) error {
	commissionRate := summary.commissionRate
	if commissionRate.IsNil() {
		commissionRate = math.LegacyZeroDec()
	}

	event := &tokenomicstypes.EventValidatorRewardDistribution{
		SessionEndBlockHeight:     summary.sessionEndHeight,
		OpReason:                  summary.opReason,
		ValidatorOperatorAddress:  summary.entry.validator.GetOperator(),
		ValidatorAccountAddress:   summary.entry.accAddr,
		CommissionRate:            commissionRate.String(),
		PoolShareUpokt:            summary.poolShare.String(),
		CommissionUpokt:           summary.commission.String(),
		SelfDelegationRewardUpokt: summary.selfDelegationReward.String(),
		DelegatorsRewardUpokt:     summary.delegatorsReward.String(),
		TotalDelegatedStakeUpokt:  summary.totalDelegatedStake.String(),
		NumDelegators:             summary.numDelegators,
	}

	return cosmostypes.UnwrapSDKContext(ctx).EventManager().EmitTypedEvent(event)
}

// buildValidatorStakes parses each validator's operator address and returns:
//  1. validatorEntries: validators with a successfully-derived account address, in
//     deterministic (account-address-ascending) order;
//  2. validatorStakeAmounts: account address -> bonded tokens, for Level 1 distribution;
//  3. totalValidatorStake: the sum of all eligible validators' bonded tokens (the Level 1
//     denominator), guaranteeing Σ poolShare_v == totalReward after LRM.
func buildValidatorStakes(
	logger cosmoslog.Logger,
	validators []stakingtypes.Validator,
) ([]validatorPoolEntry, map[string]math.Int, math.Int) {
	validatorEntries := make([]validatorPoolEntry, 0, len(validators))
	validatorStakeAmounts := make(map[string]math.Int, len(validators))
	totalValidatorStake := math.ZeroInt()

	for _, validator := range validators {
		valAddr, err := cosmostypes.ValAddressFromBech32(validator.GetOperator())
		if err != nil {
			logger.Error(fmt.Sprintf(
				"Failed to parse validator operator address %s: %v. Skipping to the next one.",
				validator.GetOperator(), err,
			))
			continue
		}

		validatorBondedTokens := validator.GetBondedTokens()
		if validatorBondedTokens.IsZero() {
			logger.Warn(fmt.Sprintf(
				"SHOULD NEVER HAPPEN: Validator %s has zero bonded tokens. Skipping to the next one.",
				validator.GetOperator(),
			))
			continue
		}

		accAddr := cosmostypes.AccAddress(valAddr).String()

		// Defense-in-depth: the staking module enforces ValAddress uniqueness in
		// the bonded set (validators are keyed by OperatorAddress), and AccAddress
		// is a 1:1 byte-equivalent of ValAddress with a different bech32 prefix.
		// In practice the duplicate branch below is consensus-impossible. The
		// guard catches any future code path that somehow injects two validators
		// with the same underlying address bytes — silently overwriting in
		// validatorStakeAmounts would (a) drop one validator's stake from the
		// Level-1 denominator and (b) double-count a single validator's bonded
		// tokens. Both outcomes are settlement-corrupting; skipping the duplicate
		// is the safer side of the unreachable branch.
		if _, exists := validatorStakeAmounts[accAddr]; exists {
			logger.Error(fmt.Sprintf(
				"SHOULD NEVER HAPPEN: duplicate validator AccAddress %s in bonded set; skipping the second occurrence (validator: %s)",
				accAddr, validator.GetOperator(),
			))
			continue
		}

		validatorEntries = append(validatorEntries, validatorPoolEntry{validator: validator, accAddr: accAddr})
		validatorStakeAmounts[accAddr] = validatorBondedTokens
		totalValidatorStake = totalValidatorStake.Add(validatorBondedTokens)
	}

	// Sort by account address (ascending) to ensure deterministic Level 2 iteration order.
	sort.Slice(validatorEntries, func(i, j int) bool {
		return validatorEntries[i].accAddr < validatorEntries[j].accAddr
	})

	return validatorEntries, validatorStakeAmounts, totalValidatorStake
}

// calculateCommission returns floor(poolShare × commissionRate), clamped to [0, poolShare].
// Commission is computed with the staking module's deterministic LegacyDec arithmetic.
func calculateCommission(poolShare math.Int, commissionRate math.LegacyDec) math.Int {
	if commissionRate.IsNil() || !commissionRate.IsPositive() {
		return math.ZeroInt()
	}
	if commissionRate.GTE(math.LegacyOneDec()) {
		return poolShare
	}
	return commissionRate.MulInt(poolShare).TruncateInt()
}

// collectValidatorDelegationStakes returns the stake map and total delegated stake for a
// single validator's delegations (including the validator's self-delegation).
// The boolean return is false if the delegation query failed.
func collectValidatorDelegationStakes(
	ctx context.Context,
	logger cosmoslog.Logger,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validator stakingtypes.Validator,
	validatorAccAddr string,
) (map[string]math.Int, math.Int, bool) {
	valAddr, err := cosmostypes.ValAddressFromBech32(validator.GetOperator())
	if err != nil {
		// Already parsed successfully in buildValidatorStakes; treat as sole-stakeholder fallback.
		return nil, math.ZeroInt(), false
	}

	delegations, err := stakingKeeper.GetValidatorDelegations(ctx, valAddr)
	if err != nil {
		logger.Warn(fmt.Sprintf(
			"SHOULD NEVER HAPPEN: Failed to get delegations for validator %s: %v. Treating validator as sole stakeholder.",
			validator.GetOperator(), err,
		))
		return nil, math.ZeroInt(), false
	}

	if len(delegations) == 0 {
		logger.Debug(fmt.Sprintf(
			"Validator %s has no delegations. Treating validator as sole stakeholder.",
			validator.GetOperator(),
		))
		return nil, math.ZeroInt(), true
	}

	stakeAmounts := make(map[string]math.Int, len(delegations))
	totalDelegatedStake := math.ZeroInt()
	for _, delegation := range delegations {
		delegatorAddr, err := cosmostypes.AccAddressFromBech32(delegation.GetDelegatorAddr())
		if err != nil {
			logger.Error(fmt.Sprintf("SHOULD NEVER HAPPEN: failed to parse delegator address %s: %v. Skipping to the next one...", delegation.GetDelegatorAddr(), err))
			continue
		}
		delegatorAddrStr := delegatorAddr.String()

		delegatedShares := delegation.GetShares()
		if delegatedShares.IsZero() {
			continue
		}

		// Convert shares to tokens using the validator's exchange rate.
		delegatedTokens := validator.TokensFromShares(delegatedShares).TruncateInt()
		if delegatedTokens.IsZero() {
			logger.Warn(fmt.Sprintf("SHOULD NEVER HAPPEN: delegator %s has zero delegated tokens but the delegated share exists. Skipping to the next one...", delegatorAddrStr))
			continue
		}

		if existing, ok := stakeAmounts[delegatorAddrStr]; ok {
			stakeAmounts[delegatorAddrStr] = existing.Add(delegatedTokens)
		} else {
			stakeAmounts[delegatorAddrStr] = delegatedTokens
		}
		totalDelegatedStake = totalDelegatedStake.Add(delegatedTokens)
	}

	return stakeAmounts, totalDelegatedStake, true
}

// addReward accumulates a reward amount for a recipient account address in place.
func addReward(rewardAmounts map[string]math.Int, addr string, amount math.Int) {
	if existing, ok := rewardAmounts[addr]; ok {
		rewardAmounts[addr] = existing.Add(amount)
	} else {
		rewardAmounts[addr] = amount
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
	// Use consolidated calculation to get reward data for all addresses
	rewardData := calculateAddressRewards(stakeAmounts, totalBondedTokens, totalRewardAmount)

	// Build reward amounts map and log details
	rewardAmounts := make(map[string]math.Int, len(rewardData))
	for _, data := range rewardData {
		rewardAmounts[data.address] = data.baseReward

		logger.Debug(fmt.Sprintf(
			"  stakeholder %s: stake=%s, base_reward=%s, fraction=%s",
			data.address,
			data.stake.String(),
			data.baseReward.String(),
			data.fraction.FloatString(6),
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

	// Calculate remainder tokens to distribute.
	//
	// Conversion note: `BigInt().Int64()` does NOT panic on overflow — it
	// silently truncates the bottom 64 bits. That truncation is acceptable here
	// because `remainderInt` is bounded by `len(stakeholders)` (at most a few
	// hundred validators + delegators on mainnet) — well below MaxInt64. If the
	// stakeholder set ever grew to billions, the LRM remainder distribution
	// algorithm would have other scaling problems long before overflow.
	remainderInt := totalRewardAmount.Sub(totalDistributedRewardAmount)
	remainder := remainderInt.BigInt().Int64()

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
	sortedAddressesByFractionDesc := sortAddressesByFractionDesc(stakeAmounts, totalBondedTokens, totalRewardAmount)
	numAddresses := int64(len(sortedAddressesByFractionDesc))

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

	for i, addrStr := range sortedAddressesByFractionDesc {
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

// queueRewardTransfers creates and queues one reward transfer per recipient in
// deterministic (account-address-ascending) order. Recipients that are validator
// operator accounts are tagged with the validator op-reason; all others use the
// delegator op-reason.
func queueRewardTransfers(
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	rewardAmounts map[string]math.Int,
	validatorAccAddresses map[string]bool,
	settlementOpReason tokenomicstypes.SettlementOpReason,
	expectedTotalReward math.Int,
) error {
	logger = logger.With("method", "queueRewardTransfers")

	// Sort recipient addresses ascending for deterministic queueing.
	sortedRecipients := make([]string, 0, len(rewardAmounts))
	for addr := range rewardAmounts {
		sortedRecipients = append(sortedRecipients, addr)
	}
	sort.Strings(sortedRecipients)

	// Use for logging purposes only
	totalDistributed := math.ZeroInt()
	numValidators := 0

	for _, addrStr := range sortedRecipients {
		rewardAmount := rewardAmounts[addrStr]
		if rewardAmount.IsZero() {
			logger.Debug(fmt.Sprintf(
				"SHOULD RARELY HAPPEN: recipient %s reward is zero, skipping",
				addrStr,
			))
			continue
		}

		totalDistributed = totalDistributed.Add(rewardAmount)
		rewardCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, rewardAmount)

		// Determine if this is a delegator or validator reward.
		isValidator := validatorAccAddresses[addrStr]
		actualRewardOpReason := settlementOpReason
		recipientType := "validator"

		if isValidator {
			numValidators++
		} else {
			// This is a delegator reward - use the delegator operation reason.
			recipientType = "delegator"
			switch settlementOpReason {
			// Mint = Burn
			case tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_VALIDATOR_REWARD_DISTRIBUTION:
				actualRewardOpReason = tokenomicstypes.SettlementOpReason_TLM_RELAY_BURN_EQUALS_MINT_DELEGATOR_REWARD_DISTRIBUTION
			// TLM Global Mint
			case tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_VALIDATOR_REWARD_DISTRIBUTION:
				actualRewardOpReason = tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_DELEGATOR_REWARD_DISTRIBUTION
			}
		}

		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         actualRewardOpReason,
			SenderModule:     tokenomicstypes.ModuleName,
			RecipientAddress: addrStr,
			Coin:             rewardCoin,
		})

		logger.Info(fmt.Sprintf(
			"queued reward transfer: %s to %s %s",
			rewardCoin.String(),
			recipientType,
			addrStr,
		))
	}

	logger.Info(fmt.Sprintf(
		"validator and delegator reward distribution complete: distributed %s to %d validators and %d total recipients",
		totalDistributed.String(),
		numValidators,
		len(sortedRecipients),
	))

	// Conservation check (telemetry only). The Largest Remainder Method is
	// mathematically conservative: the sum of all per-recipient rewards must
	// equal the input total. A drift here would indicate either a bug in the
	// LRM remainder distribution or a future code path that adds/drops
	// recipients between Step 2 and queueRewardTransfers. We log loud but do
	// NOT return an error — failing the entire settlement on a logging
	// invariant would be a worse outcome than the drift itself.
	//
	// Skipping the check when expectedTotalReward.IsZero() — for callers that
	// don't pass a meaningful total (this branch is unreachable today but the
	// guard keeps the log readable if a future caller passes ZeroInt).
	if !expectedTotalReward.IsZero() && !totalDistributed.Equal(expectedTotalReward) {
		logger.Error(fmt.Sprintf(
			"VALIDATOR REWARD CONSERVATION BREACH: distributed=%s, expected=%s, delta=%s, op_reason=%s",
			totalDistributed.String(),
			expectedTotalReward.String(),
			expectedTotalReward.Sub(totalDistributed).String(),
			settlementOpReason.String(),
		))
	}

	return nil
}
