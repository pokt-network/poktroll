package keeper

import (
	"runtime"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/proof/types"
)

// ValidateSubmittedProofs validates all proofs submitted in the block and removes
// any invalid proof from the store so that it is not included in the block.
func (k Keeper) ValidateSubmittedProofs(ctx sdk.Context) {
	logger := k.Logger().With("method", "ValidateSubmittedProofs")

	// Use an iterator to iterate over all proofs instead of fetching them all
	// at once to avoid memory issues.
	iterator := k.GetAllProofsIterator(ctx)
	defer iterator.Close()

	// Since the proofs are independent of each other, we can validate them in parallel
	// across all CPU cores to speed up the process.

	// Use a semaphore to limit the number of goroutines to the number of CPU cores.
	// This is to avoid creating too many goroutines which can lead to memory issues.
	sem := make(chan struct{}, runtime.NumCPU())

	// Use a wait group to wait for all goroutines to finish before returning.
	wg := sync.WaitGroup{}

	for ; iterator.Valid(); iterator.Next() {
		proofBz := iterator.Value()

		// Acquire a semaphore to limit the number of goroutines.
		// This will block if the sem channel is full.
		sem <- struct{}{}
		// Increment the wait group to wait for validation to finish.
		wg.Add(1)

		go func(proofBz []byte) {
			// Decrement the wait group when the goroutine finishes.
			defer wg.Done()
			// Release the semaphore after the goroutine finishes which unblocks another
			// iteration to run its goroutine.
			defer func() { <-sem }()

			var proof types.Proof
			// proofBz is not expected to fail unmarshalling since it is should have
			// passed EnsureWellFormedProof validation in MsgSubmitProof handler.
			// Panic if it fails unmarshalling.
			k.cdc.MustUnmarshal(proofBz, &proof)

			// Already validated proofs will have their ClosestMerkleProof cleared.
			// Skip already validated proofs submitted at earlier block heights of
			// the proof submission window.
			if len(proof.ClosestMerkleProof) == 0 {
				return
			}

			// Try to validate the proof and remove it if it is invalid.
			if err := k.EnsureValidProofSignaturesAndClosestPath(ctx, &proof); err != nil {
				// Remove the proof if it is invalid to save block space and trigger the
				// supplier slashing code path in the SettlePendingClaims flow.
				k.RemoveProof(ctx, proof.GetSessionHeader().GetSessionId(), proof.GetSupplierOperatorAddress())

				// TODO_MAINNET(red-0ne): Emit an invalid proof event to signal that a proof was
				// removed due to bad signatures or ClosestMerkleProof.
				// For now this could be inferred from the EventProofSubmitted+EventClaimExpired events.

				logger.Info("Removed invalid proof",
					"session_id", proof.GetSessionHeader().GetSessionId(),
					"supplier_operator_address", proof.GetSupplierOperatorAddress(),
					"error", err,
				)

				return
			}

			// Clear the ClosestMerkleProof for successfully validated proofs to:
			// 1. Save block space as the ClosestMerkleProof embeds the entire relay request and
			//    response bytes which account for the majority of the proof size.
			// 2. Mark the proof as validated to avoid re-validating it in subsequent blocks
			//    within the same proof submission window.
			proof.ClosestMerkleProof = make([]byte, 0, 0)

			// Update the proof in the store to clear the ClosestMerkleProof which makes the
			// committed block to never store the potentially large ClosestMerkleProof.
			k.UpsertProof(ctx, proof)
		}(proofBz)
	}

	// Wait for all goroutines to finish before returning.
	wg.Wait()
}
