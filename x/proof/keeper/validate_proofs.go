package keeper

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/pocket/x/proof/types"
)

// proofValidationTaskCoordinator is a helper struct to coordinate parallel proof
// validation tasks.
type proofValidationTaskCoordinator struct {
	// sem is a semaphore to limit the number of concurrent goroutines.
	sem chan struct{}

	// wg is a wait group to wait for all goroutines to finish before returning.
	wg *sync.WaitGroup

	// processedProofs is a map of supplier operator addresses to the session IDs
	// whose proofs that have been processed.
	processedProofs map[string][]string

	// numValidProofs and numInvalidProofs are counters to keep track of proof validation results.
	numValidProofs,
	numInvalidProofs uint64

	// coordinatorMu protects the coordinator fields.
	coordinatorMu *sync.Mutex
}

// numCPU caches runtime.NumCPU() to avoid being retrieved on every ValidateSubmittedProofs call.
var numCPU int

func init() {
	// Initialize the number of CPU cores available on the machine.
	numCPU = runtime.NumCPU()
}

// ValidateSubmittedProofs performs concurrent proof validation, updating claims'
// proof validation states and removing processed proofs from storage.
func (k Keeper) ValidateSubmittedProofs(ctx sdk.Context) (numValidProofs, numInvalidProofs uint64, err error) {
	logger := k.Logger().With("method", "ValidateSubmittedProofs")

	logger.Info(fmt.Sprintf("Number of CPU cores used for parallel proof validation: %d\n", numCPU))

	// Iterate over proofs using an iterator to prevent OOM issues caused by bulk fetching.
	proofIterator := k.GetAllProofsIterator(ctx)

	proofValidationCoordinator := &proofValidationTaskCoordinator{
		// Parallelize proof validation across CPU cores since they are independent from one another.
		// Use semaphores to limit concurrent goroutines and prevent memory issues.
		sem: make(chan struct{}, numCPU),
		// Use a wait group to wait for all goroutines to finish before returning.
		wg: &sync.WaitGroup{},

		processedProofs: make(map[string][]string),
		coordinatorMu:   &sync.Mutex{},
	}

	for ; proofIterator.Valid(); proofIterator.Next() {
		proofBz := proofIterator.Value()

		// Acquire a semaphore to limit the number of goroutines.
		// This will block if the sem channel is full.
		proofValidationCoordinator.sem <- struct{}{}

		// Increment the wait group to wait for proof validation to finish.
		proofValidationCoordinator.wg.Add(1)

		go k.validateProof(ctx, proofBz, proofValidationCoordinator)
	}

	// Wait for all goroutines to finish before returning.
	proofValidationCoordinator.wg.Wait()

	// Close the proof iterator before deleting the processed proofs.
	proofIterator.Close()

	// Delete all the processed proofs from the store since they are no longer needed.
	logger.Info("removing processed proofs from the store")
	for supplierOperatorAddr, processedProofs := range proofValidationCoordinator.processedProofs {
		for _, sessionId := range processedProofs {
			k.RemoveProof(ctx, sessionId, supplierOperatorAddr)
			logger.Info(fmt.Sprintf(
				"removing proof for supplier %s with session ID %s",
				supplierOperatorAddr,
				sessionId,
			))
		}
	}

	return proofValidationCoordinator.numValidProofs, proofValidationCoordinator.numInvalidProofs, nil
}

// validateProof validates a proof submitted by a supplier.
// It marks the corresponding claim as valid or invalid based on the proof validation.
// It is meant to be called concurrently by multiple goroutines to parallelize
// proof validation.
func (k Keeper) validateProof(
	ctx context.Context,
	proofBz []byte,
	coordinator *proofValidationTaskCoordinator,
) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	logger := k.Logger().With("method", "validateProof")

	// Decrement the wait group when the goroutine finishes.
	defer coordinator.wg.Done()

	// Release the semaphore after the goroutine finishes which unblocks another one.
	defer func() { <-coordinator.sem }()

	var proof types.Proof
	// proofBz is not expected to fail unmarshalling since it is should have
	// passed EnsureWellFormedProof validation in MsgSubmitProof handler.
	// Panic if it fails unmarshalling.
	// If a failure occurs, it indicates either a bug in the code or data corruption.
	// In either case, panicking is an appropriate response since both panics and
	// returning an error would halt block production.
	k.cdc.MustUnmarshal(proofBz, &proof)

	sessionHeader := proof.GetSessionHeader()
	supplierOperatorAddr := proof.GetSupplierOperatorAddress()

	logger = logger.With(
		"session_id", sessionHeader.GetSessionId(),
		"application_address", sessionHeader.GetApplicationAddress(),
		"service_id", sessionHeader.GetServiceId(),
		"session_end_height", sessionHeader.GetSessionEndBlockHeight(),
		"supplier_operator_address", supplierOperatorAddr,
	)

	// Retrieve the corresponding claim for the proof submitted so it can be
	// used in the proof validation below.
	// EnsureWellFormedProof which is called in MsgSubmitProof handler has already validated
	// that the claim referenced by the proof exists and has a matching session header.
	claim, claimFound := k.GetClaim(ctx, sessionHeader.GetSessionId(), supplierOperatorAddr)
	if !claimFound {
		// DEV_NOTE: This should never happen since EnsureWellFormedProof has already checked
		// that the proof has a corresponding claim.
		logger.Error("no claim found for the corresponding proof")
		return
	}
	logger.Debug("successfully retrieved claim")

	// Set the proof status to valid by default.
	proofStatus := types.ClaimProofStatus_VALIDATED
	// Set the invalidity reason to an empty string by default.
	invalidProofReason := ""

	if err := k.EnsureValidProofSignaturesAndClosestPath(ctx, &claim, &proof); err != nil {
		// Set the proof status to invalid.
		proofStatus = types.ClaimProofStatus_INVALID

		// Set the invalidity reason to the error message.
		invalidProofReason = err.Error()

		logger.Info(fmt.Sprintf("invalid proof due to error: %v", err))
	}
	logger.Info(fmt.Sprintf("proof checked, validation result: %s", proofStatus))

	// Create and emit an event for the proof validation result.
	eventProofValidityChecked := types.EventProofValidityChecked{
		Proof:         &proof,
		BlockHeight:   uint64(sdkCtx.BlockHeight()),
		ProofStatus:   proofStatus,
		FailureReason: invalidProofReason,
	}

	if err := sdkCtx.EventManager().EmitTypedEvent(&eventProofValidityChecked); err != nil {
		logger.Error(fmt.Sprintf("failed to emit proof validity check event due to: %v", err))
		return
	}

	// Protect the subsequent operations from concurrent access.
	coordinator.coordinatorMu.Lock()
	defer coordinator.coordinatorMu.Unlock()

	// Update the claim to reflect the validation result of the associated proof.
	//
	// It will be used later by the SettlePendingClaims routine to determine whether:
	// 1. The claim should be settled or not
	// 2. The corresponding supplier should be slashed or not
	claim.ProofValidationStatus = proofStatus
	k.UpsertClaim(ctx, claim)

	// Collect the processed proofs info to delete them after the proofIterator is closed
	// to prevent iterator invalidation.
	coordinator.processedProofs[supplierOperatorAddr] = append(
		coordinator.processedProofs[supplierOperatorAddr],
		sessionHeader.GetSessionId(),
	)

	if proofStatus == types.ClaimProofStatus_INVALID {
		// Increment the number of invalid proofs.
		coordinator.numInvalidProofs++
	} else {
		// Increment the number of valid proofs.
		coordinator.numValidProofs++
	}
}
