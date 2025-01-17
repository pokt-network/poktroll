package proof

import (
	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
)

// EndBlocker called at every block and validates all proofs submitted at the block
// height and removes any invalid proofs.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) (err error) {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	// ValidateSubmittedProofs does not return an error as it is a best-effort function.
	k.ValidateSubmittedProofs(ctx)

	return nil
}
