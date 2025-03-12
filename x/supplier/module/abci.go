package supplier

import (
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/keeper"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// EndBlocker is called every block and handles supplier related updates.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	logger := k.Logger().With("method", "EndBlocker")

	numUnbondedSuppliers, err := k.EndBlockerUnbondSuppliers(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not unbond suppliers due to error %v", err))
		return err
	}

	k.Logger().Info(fmt.Sprintf("unbonded %d suppliers", numUnbondedSuppliers))

	numSuppliersWithPrunedHistory, err := k.EndBlockerPruneSupplierServiceConfigHistory(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not prune service update history due to error %v", err))
		return err
	}

	k.Logger().Info(fmt.Sprintf("pruned service config history for %d suppliers", numSuppliersWithPrunedHistory))

	return nil
}

// BeginBlocker is called every block and handles supplier related updates that
// need to be effective at the start of the block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) error {
	// Telemetry: measure the begin-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyBeginBlocker)

	logger := k.Logger().With("method", "BeginBlocker")

	// Service activation occurs in BeginBlocker rather than EndBlocker for two main reasons:
	//
	// 1. External State Consistency: When offchain clients query the suppliers at activation
	//    block (height N), they should see the newly activated service configurations.
	//    * If the activation occurred in the EndBlocker of block N-1, clients would
	//      observe a mismatch between the activation height and when the configurations
	//      actually take effect.
	//    * If the activation occurred in the EndBlocker of block N, offchain clients will
	//      have a consistent view but the internal state consistency will not be satisfied.
	//
	// 2. Internal State Consistency: All transactions within the activation block must
	//    execute against a consistent set of service configurations. Activating in
	//    BeginBlocker ensures all state transitions in block N operate with the new
	//    configurations, whereas activation in EndBlocker would build the block
	//    on the old configurations.
	numSuppliersWithActivatedServices, err := k.BeginBlockerActivateSupplierServices(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not activate services for suppliers due to error %v", err))
		return err
	}

	logger.Info(fmt.Sprintf("activated services for %d suppliers", numSuppliersWithActivatedServices))

	return nil
}
