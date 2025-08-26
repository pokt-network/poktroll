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

	// Sort delegations deterministically to ensure consistent remainder distribution
	// Primary sort: by delegation shares (ascending - least shares first)
	// Secondary sort: by delegator address (lexicographical for ties)
	sort.Slice(delegations, func(i, j int) bool {
		sharesI := delegations[i].Shares
		sharesJ := delegations[j].Shares

		// If shares are equal, sort by delegator address (lexicographical)
		if sharesI.Equal(sharesJ) {
			return delegations[i].DelegatorAddress < delegations[j].DelegatorAddress
		}

		// Otherwise, sort by shares (ascending - least shares first)
		return sharesI.LT(sharesJ)
	})

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

	// Add any remainder to the delegator with the least shares (first in sorted order)
	// This ensures deterministic remainder distribution regardless of execution order
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

// distributeRewardsToAllValidatorsAndDelegatesByStakeWeight distributes rewards to all bonded validators
// and their delegators proportionally based on their staking weight. This function encapsulates the
// common logic used by multiple TLMs to distribute validator rewards.
//
// This function:
// 1. Gets all bonded validators sorted by voting power
// 2. Calculates total bonded tokens across all validators
// 3. Distributes rewards proportionally to each validator based on their stake weight
// 4. Calls distributeValidatorRewardsToStakeholders for each validator to handle commission and delegator distribution
// 5. Handles edge cases (no validators, zero stake) by returning nil (rewards go to DAO)
func distributeRewardsToAllValidatorsAndDelegatesByStakeWeight(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	totalRewardAmount math.Int,
	validatorCommissionOpReason tokenomicstypes.SettlementOpReason,
	delegatorRewardOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeRewardsToAllValidatorsAndDelegatesByStakeWeight",
		"total_reward_amount", totalRewardAmount,
	)

	if totalRewardAmount.IsZero() {
		logger.Debug("total reward amount is zero, skipping distribution")
		return nil
	}

	totalRewardCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, totalRewardAmount)

	// Get all bonded validators sorted by voting power
	validators, err := stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to retrieve bonded validators for reward distribution: %v", err))
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error getting bonded validators: %v", err)
	}

	if len(validators) == 0 {
		logger.Warn("no bonded validators found for proposer reward distribution - rewards will go to DAO")
		// Rewards will be included in DAO allocation since no validators to distribute to
		return nil
	}

	// Calculate total bonded tokens across all validators
	totalBondedTokens := math.ZeroInt()
	validatorsWithStake := 0
	for _, validator := range validators {
		bondedTokens := validator.GetBondedTokens()
		if bondedTokens.IsPositive() {
			totalBondedTokens = totalBondedTokens.Add(bondedTokens)
			validatorsWithStake++
		}
	}

	if totalBondedTokens.IsZero() {
		logger.Warn("total bonded tokens is zero across all validators, skipping proposer reward distribution - rewards will go to DAO")
		// Rewards will be included in DAO allocation since no stake to distribute based on
		return nil
	}

	logger.Info(fmt.Sprintf("distributing (%v) to %d validators with stake (total: %s tokens bonded)",
		totalRewardCoin, validatorsWithStake, totalBondedTokens))

	// Sort validators deterministically to ensure consistent remainder distribution
	// Primary sort: by bonded tokens (ascending - least stake first)
	// Secondary sort: by operator address (lexicographical for ties)
	sort.Slice(validators, func(i, j int) bool {
		stakeI := validators[i].GetBondedTokens()
		stakeJ := validators[j].GetBondedTokens()

		// If stakes are equal, sort by operator address (lexicographical)
		if stakeI.Equal(stakeJ) {
			return validators[i].GetOperator() < validators[j].GetOperator()
		}

		// Otherwise, sort by stake (ascending - least stake first)
		return stakeI.LT(stakeJ)
	})

	// Calculate proportional shares for all validators and track total distributed
	validatorShares := make([]math.Int, len(validators))
	totalDistributed := math.ZeroInt()
	distributedValidators := 0

	for i, validator := range validators {
		// Skip validators with zero stake
		validatorBondedTokens := validator.GetBondedTokens()
		if validatorBondedTokens.IsZero() {
			logger.Debug(fmt.Sprintf("skipping validator %s with zero bonded tokens", validator.GetOperator()))
			validatorShares[i] = math.ZeroInt()
			continue
		}

		// Calculate proportional share: (validator_tokens / total_tokens) * total_reward_amount
		validatorShare := totalRewardAmount.Mul(validatorBondedTokens).Quo(totalBondedTokens)
		validatorShares[i] = validatorShare
		totalDistributed = totalDistributed.Add(validatorShare)
		distributedValidators++
	}

	// Distribute any remainder to the validator with the least stake (first in sorted order)
	// This ensures deterministic remainder distribution regardless of TLM execution order
	remainder := totalRewardAmount.Sub(totalDistributed)
	if !remainder.IsZero() && distributedValidators > 0 {
		// Find first validator with non-zero stake (they are sorted by stake ascending)
		for i := 0; i < len(validators); i++ {
			if !validatorShares[i].IsZero() {
				validatorShares[i] = validatorShares[i].Add(remainder)
				logger.Debug(fmt.Sprintf("allocated remainder (%v) to validator %s with least stake (%s tokens)",
					cosmostypes.NewCoin(pocket.DenomuPOKT, remainder),
					validators[i].GetOperator(),
					validators[i].GetBondedTokens()))
				break
			}
		}
	}

	// Distribute rewards to each validator
	actuallyDistributed := 0
	for i, validator := range validators {
		validatorShare := validatorShares[i]

		if validatorShare.IsZero() {
			logger.Debug(fmt.Sprintf("validator %s calculated share is zero, skipping", validator.GetOperator()))
			continue
		}

		// Distribute rewards directly to validator and delegators using ModToAcctTransfer
		if err := distributeValidatorRewardsToStakeholders(
			ctx,
			logger,
			result,
			stakingKeeper,
			&validators[i],
			validatorShare,
			validatorCommissionOpReason,
			delegatorRewardOpReason,
		); err != nil {
			logger.Error(fmt.Sprintf("failed to distribute rewards to validator %s stakeholders: %v", validator.GetOperator(), err))
			return tokenomicstypes.ErrTokenomicsTLMInternal.Wrapf("error distributing rewards to validator %s stakeholders: %v", validator.GetOperator(), err)
		}

		actuallyDistributed++

		// Get validator bonded tokens for logging
		validatorBondedTokens := validator.GetBondedTokens()
		logger.Debug(fmt.Sprintf("distributed (%v) to validator %s and delegators (stake: %s/%s, weight: %.4f%%)",
			cosmostypes.NewCoin(pocket.DenomuPOKT, validatorShare),
			validator.GetOperator(),
			validatorBondedTokens,
			totalBondedTokens,
			float64(validatorBondedTokens.Int64())/float64(totalBondedTokens.Int64())*100))
	}

	if actuallyDistributed == 0 {
		logger.Error("no validators received rewards despite having stake - this should not happen")
		return tokenomicstypes.ErrTokenomicsTLMInternal.Wrap("no validators received rewards despite having stake")
	}

	logger.Info(fmt.Sprintf("successfully distributed (%v) to %d validators and their delegators using ModToAcctTransfer", totalRewardCoin, actuallyDistributed))
	return nil
}
