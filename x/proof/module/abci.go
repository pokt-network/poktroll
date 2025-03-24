package proof

import (
	"fmt"

	cosmostelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/proof/keeper"
	"github.com/pokt-network/poktroll/x/proof/types"
)

// EndBlocker is called at every block and handles proof-related operations.
func EndBlocker(ctx sdk.Context, k keeper.Keeper) (err error) {
	// Telemetry: measure the end-block execution time following standard cosmos-sdk practices.
	defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)

	logger := k.Logger().With("method", "EndBlocker")

	// Iterates through all proofs submitted in this block and:
	//   1. Updates the proof validation status in the associated claim
	//   2. Removes all processed proofs from onchain state
	numValidProofs, numInvalidProofs, err := k.ValidateSubmittedProofs(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("could not validate submitted proofs due to error %v", err))
		return err
	}

	logger.Info(fmt.Sprintf(
		"checked %d proofs: %d valid, %d invalid",
		numValidProofs+numInvalidProofs,
		numValidProofs,
		numInvalidProofs,
	))

	return nil
}
