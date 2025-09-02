package token_logic_module

import (
	"context"
	"fmt"
	"math/big"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

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

// distributeValidatorRewards distributes session settlement rewards to all bonded
// validators proportionally based on their staking weight. This implements pure
// stake-weighted distribution without any commission calculations, as these are
// session settlement rewards (not consensus rewards).
//
// The distribution formula is:
// validatorReward = totalValidatorReward × (validatorStake / totalBondedStake)
func distributeValidatorRewards(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	stakingKeeper tokenomicstypes.StakingKeeper,
	validatorRewardAmount math.Int,
	settlementOpReason tokenomicstypes.SettlementOpReason,
) error {
	logger = logger.With(
		"method", "distributeValidatorRewards",
		"session_id", result.GetSessionId(),
		"total_reward_amount", validatorRewardAmount,
	)

	if validatorRewardAmount.IsZero() {
		logger.Debug("validator reward amount is zero, skipping distribution")
		return nil
	}

	// Get all bonded validators
	validators, err := stakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"failed to get bonded validators: %v", err,
		)
	}

	if len(validators) == 0 {
		logger.Warn("no bonded validators found, skipping validator reward distribution")
		return nil
	}

	// Calculate total bonded tokens across all validators
	totalBondedTokens := math.ZeroInt()
	for _, validator := range validators {
		totalBondedTokens = totalBondedTokens.Add(validator.GetBondedTokens())
	}

	if totalBondedTokens.IsZero() {
		logger.Warn("total bonded tokens is zero, skipping validator reward distribution")
		return nil
	}

	logger.Info(fmt.Sprintf(
		"distributing %s to %d validators based on stake weight (total bonded: %s)",
		validatorRewardAmount.String(),
		len(validators),
		totalBondedTokens.String(),
	))

	// Calculate and distribute rewards to each validator based on stake weight
	totalDistributed := math.ZeroInt()

	for i, validator := range validators {
		validatorStake := validator.GetBondedTokens()

		// Create fraction: validatorStake / totalBondedTokens
		stakeRatio := new(big.Rat).SetFrac(
			validatorStake.BigInt(),
			totalBondedTokens.BigInt(),
		)

		// Multiply by total reward amount: reward × (validatorStake / totalStake)
		validatorRewardRat := new(big.Rat).Mul(
			stakeRatio,
			new(big.Rat).SetInt(validatorRewardAmount.BigInt()),
		)

		// Convert back to integer (truncating decimal portion)
		validatorReward := math.NewIntFromBigInt(new(big.Int).Quo(
			validatorRewardRat.Num(),
			validatorRewardRat.Denom(),
		))

		// Handle remainder: add any leftover to the last validator's reward
		if i == len(validators)-1 {
			remainder := validatorRewardAmount.Sub(totalDistributed.Add(validatorReward))
			validatorReward = validatorReward.Add(remainder)
		}

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

		logger.Info(fmt.Sprintf(
			"queued reward transfer: %s to validator %s (stake: %s, share: %s%%)",
			validatorRewardCoin.String(),
			validator.GetOperator(),
			validatorStake.String(),
			new(big.Rat).SetFrac(
				validatorStake.BigInt(),
				totalBondedTokens.BigInt(),
			).FloatString(2),
		))
	}

	logger.Info(fmt.Sprintf(
		"validator reward distribution complete: distributed %s of %s uPOKT to %d validators",
		totalDistributed.String(),
		validatorRewardAmount.String(),
		len(validators),
	))

	return nil
}
