package token_logic_module

// This file contains the business logic necessary to distribute rewards to suppliesr
// and their shareholders.

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// requiredRevSharePercentageSum mirrors the constant of the same name in
// `x/shared/types/service_configs.go` and represents the invariant enforced by
// `ValidateServiceRevShare` on `MsgStakeSupplier`. Settlement re-validates the
// same invariant defensively to catch state-level violations (notably the
// pre-v0.1.34 duplicate-revshare migration path which sums duplicate
// percentages without re-validating the merged total).
const requiredRevSharePercentageSum = uint64(100)

// GetSupplierShareholderAmountMap calculates the amount of uPOKT to distribute to each revenue
// shareholder based on the rev share percentage of the service.
//
// It returns a map of the shareholder address to the amount of uPOKT to distribute.
// The first shareholder gets any remainder resulting from the integer division.
//
// An error is returned when the input revshare list is invalid for distribution:
//
//   - any nil entry (corrupted state);
//   - any duplicate recipient address (the underlying map would silently
//     overwrite the first occurrence — even if the percentages sum to 100,
//     letting the first occurrence's amount drop on the floor is data loss);
//   - the sum of percentages != 100 (chain-validation perimeter hole — migration
//     output can produce sum > 100 from duplicate merges; sum < 100 from
//     hand-written state);
//   - a negative remainder after distribution (defensive overflow guard:
//     unreachable when the sum-check above passes, but kept as a belt-and-
//     suspenders to prevent `NewCoin` panicking on a negative amount).
//
// On any error, the caller is expected to fall back to the supplier's
// `owner_address` and emit `EventSupplierRevShareFallbackDistribution`. Returning
// here MUST NOT halt the chain.
//
// DEV_NOTE: Exposed publicly for testing purposes.
func GetSupplierShareholderAmountMap(
	serviceRevShare []*sharedtypes.ServiceRevenueShare,
	amountToDistribute math.Int,
) (shareAmountMap map[string]math.Int, err error) {
	if len(serviceRevShare) == 0 {
		return nil, fmt.Errorf("empty revshare list")
	}

	// Pre-flight validation. Reject EVERY invalid shape before any math so the
	// caller can route to the owner-fallback path cleanly. Validating here
	// (instead of relying on the caller) keeps the contract narrow: "this
	// function either returns a valid distribution that sums to amountToDistribute
	// across unique recipients, or it returns an error".
	seen := make(map[string]struct{}, len(serviceRevShare))
	sumPct := uint64(0)
	for _, rs := range serviceRevShare {
		if rs == nil {
			return nil, fmt.Errorf("nil revshare entry encountered")
		}
		if _, duplicate := seen[rs.Address]; duplicate {
			return nil, fmt.Errorf("duplicate revshare recipient address: %q", rs.Address)
		}
		seen[rs.Address] = struct{}{}
		sumPct += rs.RevSharePercentage
	}
	if sumPct != requiredRevSharePercentageSum {
		return nil, fmt.Errorf(
			"revshare percentage sum %d != required %d", sumPct, requiredRevSharePercentageSum,
		)
	}

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

	// Add any remainder to the first shareholder. Belt-and-suspenders: when
	// sumPct == 100 holds and each per-share is a floor of (amount * pct/100),
	// the sum of floors is <= amount, so the remainder is non-negative by
	// construction. The guard below catches any future code path that bypasses
	// the sum-check above — `NewCoin` panics on negative amount and would halt
	// the chain at settlement.
	firstShareholder := serviceRevShare[0]
	remainder := amountToDistribute.Sub(totalDistributed)
	if remainder.IsNegative() {
		return nil, fmt.Errorf(
			"negative remainder %s after distribution (totalDistributed=%s > amountToDistribute=%s); refusing to risk NewCoin panic",
			remainder.String(), totalDistributed.String(), amountToDistribute.String(),
		)
	}
	shareAmountMap[firstShareholder.Address] = shareAmountMap[firstShareholder.Address].Add(remainder)

	return shareAmountMap, nil
}

// distributeSupplierRewardsToShareholders distributes the supplier rewards to its
// shareholders based on the rev share percentage of the supplier service config.
//
// If the configured RevShare list cannot be used (see `GetSupplierShareholderAmountMap`
// for the rejection cases), the full `amountToDistribute` is paid to the
// supplier's `owner_address` instead and `EventSupplierRevShareFallbackDistribution`
// is emitted for indexer/operator visibility. This rescues the chain from a
// `NewCoin` panic on a negative share AND keeps the supplier's revenue flowing
// to the proto-level owner until the operator restakes with clean revshare.
func distributeSupplierRewardsToShareholders(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	settlementOpReason tokenomicstypes.SettlementOpReason,
	supplier *sharedtypes.Supplier,
	serviceId string,
	amountToDistribute math.Int,
) error {
	logger = logger.With(
		"method", "distributeSupplierRewardsToShareholders",
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

	// Compute the deduplicated, sum-validated shareholder distribution. On any
	// invalid shape, fall through to the owner-fallback branch below — do NOT
	// propagate the error up (which would halt the chain at settlement).
	shareAmountMap, distributeErr := GetSupplierShareholderAmountMap(serviceRevShares, amountToDistribute)
	if distributeErr != nil {
		return payRevShareFallbackToOwner(
			ctx, logger, result, settlementOpReason, supplier, serviceId, amountToDistribute,
			serviceRevShares, distributeErr,
		)
	}

	sortedAddresses := make([]string, 0, len(shareAmountMap))
	for addr := range shareAmountMap {
		sortedAddresses = append(sortedAddresses, addr)
	}
	sort.Strings(sortedAddresses)

	for _, address := range sortedAddresses {
		shareAmount := shareAmountMap[address]

		// Don't queue zero amount transfer operations.
		if shareAmount.IsZero() {
			// DEV_NOTE: This should never happen, but it mitigates a chain halt if it does.
			logger.Warn(fmt.Sprintf("zero shareAmount for service rev share address %q", address))
			continue
		}

		// Queue the sending of the newley minted uPOKT from the supplier module
		// account to the supplier's shareholders.
		shareAmountCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, shareAmount)
		result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
			OpReason:         settlementOpReason,
			SenderModule:     suppliertypes.ModuleName,
			RecipientAddress: address,
			Coin:             shareAmountCoin,
		})

		logger.Info(fmt.Sprintf("operation queued: send %s from the supplier module to the supplier shareholder with address %q", shareAmountCoin, supplier.GetOperatorAddress()))
	}

	logger.Info(fmt.Sprintf("operation queued: distribute %d uPOKT to supplier %q shareholders", amountToDistribute, supplier.GetOperatorAddress()))

	return nil
}

// payRevShareFallbackToOwner queues the full supplier slice to `supplier.OwnerAddress`
// and emits EventSupplierRevShareFallbackDistribution. Used when the configured
// RevShare list cannot be used (see `GetSupplierShareholderAmountMap`).
//
// Returns an error only if the owner address is missing — in that case the claim
// cannot pay anyone safely and the caller should surface this as a faulty claim
// (which settlement handles without halting). For a normally-staked supplier this
// branch is unreachable: the owner_address is required at staking time and the
// proto field stays populated for the supplier's entire lifecycle.
func payRevShareFallbackToOwner(
	ctx context.Context,
	logger cosmoslog.Logger,
	result *tokenomicstypes.ClaimSettlementResult,
	settlementOpReason tokenomicstypes.SettlementOpReason,
	supplier *sharedtypes.Supplier,
	serviceId string,
	amountToDistribute math.Int,
	configuredRevShares []*sharedtypes.ServiceRevenueShare,
	rejectionErr error,
) error {
	if supplier.GetOwnerAddress() == "" {
		return tokenomicstypes.ErrTokenomicsConstraint.Wrapf(
			"supplier %q has empty owner address; cannot pay revshare fallback (root cause: %v)",
			supplier.GetOperatorAddress(), rejectionErr,
		)
	}

	// Best-effort observability: sum the raw revshare percentages so the indexer
	// gets the same number the operator should see in their config. Ignores nil
	// entries (avoids panicking when this branch is reached because of a nil).
	observedSum := uint64(0)
	for _, rs := range configuredRevShares {
		if rs == nil {
			continue
		}
		observedSum += rs.RevSharePercentage
	}

	logger.Error(fmt.Sprintf(
		"supplier %q revshare config invalid (%v); paying full amount %s to owner %q as fallback",
		supplier.GetOperatorAddress(), rejectionErr, amountToDistribute.String(), supplier.GetOwnerAddress(),
	))

	fallbackCoin := cosmostypes.NewCoin(pocket.DenomuPOKT, amountToDistribute)
	result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
		OpReason:         settlementOpReason,
		SenderModule:     suppliertypes.ModuleName,
		RecipientAddress: supplier.GetOwnerAddress(),
		Coin:             fallbackCoin,
	})

	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	fallbackEvent := &tokenomicstypes.EventSupplierRevShareFallbackDistribution{
		SupplierOperatorAddress: supplier.GetOperatorAddress(),
		SupplierOwnerAddress:    supplier.GetOwnerAddress(),
		ServiceId:               serviceId,
		SessionEndBlockHeight:   result.GetSessionEndHeight(),
		Amount:                  fallbackCoin.String(),
		OpReason:                settlementOpReason,
		ObservedSumPercentage:   observedSum,
		Reason:                  rejectionErr.Error(),
	}
	if emitErr := sdkCtx.EventManager().EmitTypedEvent(fallbackEvent); emitErr != nil {
		// Logging only — the fallback transfer was already queued above so the
		// claim still settles. The missing event reduces observability but does
		// not corrupt state.
		logger.Error(fmt.Sprintf(
			"failed to emit EventSupplierRevShareFallbackDistribution for supplier %q: %v",
			supplier.GetOperatorAddress(), emitErr,
		))
	}

	return nil
}
