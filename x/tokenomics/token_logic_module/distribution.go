package token_logic_module

import (
	"errors"
	"fmt"
	"math/big"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// distributeSupplierRewardsToShareHolders distributes the supplier rewards to its
// shareholders based on the rev share percentage of the supplier service config.
func distributeSupplierRewardsToShareHolders(
	logger cosmoslog.Logger,
	result *PendingSettlementResult,
	tokenLogicModule TokenLogicModule,
	supplier *sharedtypes.Supplier,
	serviceId string,
	amountToDistribute uint64,
) error {
	logger = logger.With("method", "distributeSupplierRewardsToShareHolders")

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
	// TODO_MAINNET(@red-0ne): Double check this doesn't happen.
	if serviceRevShares == nil {
		return tokenomicstypes.ErrTokenomicsSupplierRevShareFailed.Wrapf(
			"service %q not found for supplier %v",
			serviceId,
			supplier,
		)
	}

	// NOTE: Use the serviceRevShares slice to iterate through the serviceRevSharesMap deterministically.
	var errs error
	shareAmountMap := GetShareAmountMap(serviceRevShares, amountToDistribute)
	for _, revShare := range serviceRevShares {
		shareAmount := shareAmountMap[revShare.GetAddress()]

		// TODO_TECHDEBT(@red-0ne): Refactor to reuse the sendRewardsToAccount helper here.
		shareAmountCoin := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(int64(shareAmount)))

		shareHolderAccAddr, err := cosmostypes.AccAddressFromBech32(revShare.GetAddress())
		if err != nil {
			errs = errors.Join(errs, err)
		}

		// Send the newley minted uPOKT from the supplier module account
		// to the supplier's shareholders.
		result.AppendModToAcctTransfer(ModToAcctTransfer{
			TLMName:          tokenLogicModule,
			SenderModule:     suppliertypes.ModuleName,
			RecipientAddress: shareHolderAccAddr,
			Coin:             shareAmountCoin,
		})

		// TODO_IN_THIS_COMMIT: move...
		logger.Info(fmt.Sprintf("sent %s from the supplier module to the supplier shareholder with address %q", shareAmountCoin, supplier.GetOperatorAddress()))
	}

	// TODO_IN_THIS_COMMIT: move...
	logger.Info(fmt.Sprintf("distributed %d uPOKT to supplier %q shareholders", amountToDistribute, supplier.GetOperatorAddress()))

	return errs
}

// GetShareAmountMap calculates the amount of uPOKT to distribute to each revenue
// shareholder based on the rev share percentage of the service.
// It returns a map of the shareholder address to the amount of uPOKT to distribute.
// The first shareholder gets any remainder due to floating point arithmetic.
// NB: It is publically exposed to be used in the tests.
func GetShareAmountMap(
	serviceRevShare []*sharedtypes.ServiceRevenueShare,
	amountToDistribute uint64,
) (shareAmountMap map[string]uint64) {
	totalDistributed := uint64(0)
	shareAmountMap = make(map[string]uint64, len(serviceRevShare))
	for _, revShare := range serviceRevShare {
		// TODO_MAINNET: Consider using fixed point arithmetic for deterministic results.
		sharePercentageFloat := big.NewFloat(float64(revShare.RevSharePercentage) / 100)
		amountToDistributeFloat := big.NewFloat(float64(amountToDistribute))
		shareAmount, _ := big.NewFloat(0).Mul(amountToDistributeFloat, sharePercentageFloat).Uint64()
		shareAmountMap[revShare.Address] = shareAmount
		totalDistributed += shareAmount
	}

	// Add any remainder due to floating point arithmetic to the first shareholder.
	remainder := amountToDistribute - totalDistributed
	shareAmountMap[serviceRevShare[0].Address] += remainder

	return shareAmountMap
}
