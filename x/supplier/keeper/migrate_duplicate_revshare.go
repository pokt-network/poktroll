package keeper

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// DeduplicateSupplierRevShareAddresses iterates all suppliers and merges
// duplicate rev share addresses in their service config history by summing
// percentages for the same address.
//
// Returns the count of modified suppliers and any error.
func (k Keeper) DeduplicateSupplierRevShareAddresses(ctx context.Context) (int, error) {
	logger := cosmostypes.UnwrapSDKContext(ctx).Logger().With("method", "DeduplicateSupplierRevShareAddresses")

	suppliers := k.GetAllSuppliers(ctx)
	modifiedCount := 0

	for _, supplier := range suppliers {
		modified := deduplicateSupplierConfigHistory(logger, &supplier)
		if modified {
			k.SetAndIndexDehydratedSupplier(ctx, supplier)
			modifiedCount++
			logger.Info(fmt.Sprintf("deduplicated rev share addresses for supplier %s", supplier.OperatorAddress))
		}
	}

	return modifiedCount, nil
}

// deduplicateSupplierConfigHistory checks and fixes duplicate rev share
// addresses in a supplier's service config history. Returns true if any
// modifications were made.
func deduplicateSupplierConfigHistory(logger cosmoslog.Logger, supplier *sharedtypes.Supplier) bool {
	modified := false

	for _, configUpdate := range supplier.ServiceConfigHistory {
		if configUpdate == nil || configUpdate.Service == nil {
			continue
		}

		revShares := configUpdate.Service.RevShare
		if !hasDuplicateRevShareAddresses(revShares) {
			continue
		}

		configUpdate.Service.RevShare = mergeRevShareDuplicates(revShares)
		modified = true

		logger.Info(fmt.Sprintf(
			"merged duplicate rev share addresses for supplier %s service %s",
			supplier.OperatorAddress, configUpdate.Service.ServiceId,
		))
	}

	return modified
}

// hasDuplicateRevShareAddresses returns true if any address appears more than
// once in the rev share list.
func hasDuplicateRevShareAddresses(revShares []*sharedtypes.ServiceRevenueShare) bool {
	seen := make(map[string]struct{}, len(revShares))
	for _, rs := range revShares {
		if _, exists := seen[rs.Address]; exists {
			return true
		}
		seen[rs.Address] = struct{}{}
	}
	return false
}

// mergeRevShareDuplicates merges entries with the same address by summing their
// percentages. The order of the deduplicated list follows the first occurrence
// of each address in the original list.
func mergeRevShareDuplicates(revShares []*sharedtypes.ServiceRevenueShare) []*sharedtypes.ServiceRevenueShare {
	merged := make(map[string]uint64, len(revShares))
	order := make([]string, 0, len(revShares))

	for _, rs := range revShares {
		if _, exists := merged[rs.Address]; !exists {
			order = append(order, rs.Address)
		}
		merged[rs.Address] += rs.RevSharePercentage
	}

	result := make([]*sharedtypes.ServiceRevenueShare, 0, len(order))
	for _, addr := range order {
		result = append(result, &sharedtypes.ServiceRevenueShare{
			Address:            addr,
			RevSharePercentage: merged[addr],
		})
	}

	return result
}
