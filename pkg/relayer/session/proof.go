package session

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// submitProofs maps over the given claimedSessions observable.
// For each session batch, it:
// 1. Starts async processing to wait for the appropriate block height
// 2. Processes proofs asynchronously without blocking the pipeline
// 3. Maps errors to a new observable and logs them
// It DOES NOT BLOCK as async processing happens in separate goroutines.
func (rs *relayerSessionsManager) submitProofs(
	ctx context.Context,
	supplierClient client.SupplierClient,
	claimedSessionsObs observable.Observable[[]relayer.SessionTree],
) {
	failedSubmitProofsSessionsObs, failedSubmitProofsSessionsPublishCh :=
		channel.NewObservable[[]relayer.SessionTree]()

	// FIX: Async Proofs Pipeline - DECOUPLED PROCESSING
	// PROBLEM: The original pipeline blocked on waitForBlock operations
	// SOLUTION: Create separate observables for async proof processing
	//
	// Create a new observable for sessions ready to be proven
	// This will be populated asynchronously by processProofsAsync
	proofReadySessionsObs, proofReadySessionsPublishCh :=
		channel.NewObservable[[]relayer.SessionTree]()

	// Start async processing for each batch of claimed sessions
	// This immediately returns and processes in the background
	channel.ForEach(
		ctx, claimedSessionsObs,
		rs.mapStartAsyncProofProcessing(
			supplierClient,
			proofReadySessionsPublishCh,
			failedSubmitProofsSessionsPublishCh,
		),
	)

	// Map proofReadySessionsObs to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// proof has been submitted or an error has been encountered, respectively.
	eitherProvenSessionsObs := channel.Map(
		ctx, proofReadySessionsObs,
		rs.newMapProveSessionsFn(supplierClient, failedSubmitProofsSessionsPublishCh),
	)

	logging.LogErrors(ctx, filter.EitherError(ctx, eitherProvenSessionsObs))

	// Delete failed session trees so they don't get proven again.
	channel.ForEach(ctx, failedSubmitProofsSessionsObs, rs.deleteSessionTrees)
}

// mapStartAsyncProofProcessing returns a ForEachFn that starts async proof processing
// for each batch of sessions. It immediately returns to avoid blocking the pipeline.
func (rs *relayerSessionsManager) mapStartAsyncProofProcessing(
	supplierClient client.SupplierClient,
	proofReadySessionsPublishCh chan<- []relayer.SessionTree,
	failedSubmitProofsSessionsPublishCh chan<- []relayer.SessionTree,
) channel.ForEachFn[[]relayer.SessionTree] {
	return func(ctx context.Context, sessionTrees []relayer.SessionTree) {
		// FIX: Non-blocking Async Start
		// Start async processing in a separate goroutine to immediately return
		// This prevents the observable pipeline from blocking
		go rs.processProofsAsync(ctx, sessionTrees, proofReadySessionsPublishCh, failedSubmitProofsSessionsPublishCh)
	}
}

// waitForEarliestSubmitProofsHeightAndGenerateProofs calculates and waits for
// (blocking until) the earliest block height, allowed by the protocol, at which
// proofs can be submitted for a session number which were claimed at createClaimHeight.
// It is calculated relative to createClaimHeight using onchain governance parameters
// and randomized input.
func (rs *relayerSessionsManager) waitForEarliestSubmitProofsHeightAndGenerateProofs(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
	failedSubmitProofsSessionsCh chan<- []relayer.SessionTree,
) []relayer.SessionTree {
	// Guard against empty sessionTrees to prevent index out of bounds errors
	if len(sessionTrees) == 0 {
		rs.logger.Warn().Msg("⚠️ Received empty session trees array - no sessions to process")
		return nil
	}

	// Given the sessionTrees are grouped by their sessionEndHeight, we can use the
	// first one from the group to calculate the earliest height for proof submission.
	sessionEndHeight := sessionTrees[0].GetSessionHeader().GetSessionEndBlockHeight()
	supplierOperatorAddr := sessionTrees[0].GetSupplierOperatorAddress()

	logger := rs.logger.With("session_end_height", sessionEndHeight)

	// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
	// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
	// to get the most recently (asynchronously) observed (and cached) value.
	// TODO_MAINNET(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
	// we should be using the value that the params had for the session which includes queryHeight.
	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("❌️ Failed to retrieve shared network parameters. ❗Check node connectivity. ❗Unable to calculate proof timing, which may prevent rewards and cause slashing.")
		failedSubmitProofsSessionsCh <- sessionTrees
		return nil
	}

	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, sessionEndHeight)

	// we wait for proofWindowOpenHeight to be received before proceeding since we need
	// its hash to seed the pseudo-random number generator for the proof submission
	// distribution (i.e. earliestSupplierProofCommitHeight).
	logger = logger.With("proof_window_open_height", proofWindowOpenHeight)
	logger.Info().Msgf(
		"⏱️ Waiting for network-defined proof window to open at block height %d before submitting proofs",
		proofWindowOpenHeight,
	)

	proofsWindowOpenBlock := rs.waitForBlock(ctx, proofWindowOpenHeight)
	if proofsWindowOpenBlock == nil {
		// Ignore this failure during shutdown:
		// - When `stopping == true`, context cancellations and observable failures are expected.
		// - Avoid interpreting them as session failures, to ensure session trees are persisted.
		//
		// In normal operation (`stopping == false`), this is a critical error and must be handled.
		if !rs.stopping.Load() {
			logger.Error().Msg("❌️ Failed to observe required block for proof timing. ❗Check node connectivity and sync status. ❗These session proofs cannot be processed, rewards may be lost and supplier may be slashed.")
			failedSubmitProofsSessionsCh <- sessionTrees
		}

		return nil
	}

	// Get the earliest proof commit height for this supplier.
	earliestSupplierProofsCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		sessionEndHeight,
		proofsWindowOpenBlock.Hash(),
		supplierOperatorAddr,
	)

	logger = logger.With("earliest_supplier_proof_commit_height", earliestSupplierProofsCommitHeight)
	logger.Info().Msgf(
		"⌛ Waiting for the assigned proof submission timing at block %d.",
		earliestSupplierProofsCommitHeight,
	)

	// earliestSupplierProofsCommitHeight - 1 is the block that will have its hash
	// used as the source of entropy for all the session trees in that batch,
	// waiting for it to be received before proceeding.
	proofPathSeedBlockHeight := earliestSupplierProofsCommitHeight - 1
	proofPathSeedBlock := rs.waitForBlock(ctx, proofPathSeedBlockHeight)

	logger = logger.With("proof_path_seed_block", fmt.Sprintf("%x", proofPathSeedBlock.Hash()))
	logger.Info().Msg(
		"🔭 Successfully observed proof path seed block. Using block hash for deterministic proof path generation.",
	)

	successProofs, failedProofs := rs.proveClaims(ctx, sessionTrees, proofPathSeedBlock)
	failedSubmitProofsSessionsCh <- failedProofs

	if len(successProofs) > 0 {
		logger.Info().Msgf(
			"🚀 Proof generation phase complete: %d sessions ready for onchain submission",
			len(successProofs),
		)
	}

	if len(failedProofs) > 0 {
		logger.Warn().Msgf(
			"⚠️ Proof generation failed for %d sessions. ❗Check storage health and data integrity.",
			len(failedProofs),
		)
	}

	return successProofs
}

// newMapProveSessionsFn returns a new MapFn that submits proofs on the given
// session number. Any session which encounters errors while submitting a proof
// is sent on the failedSubmitProofSessions channel.
func (rs *relayerSessionsManager) newMapProveSessionsFn(
	supplierClient client.SupplierClient,
	failedSubmitProofSessionsCh chan<- []relayer.SessionTree,
) channel.MapFn[[]relayer.SessionTree, either.SessionTrees] {
	return func(
		ctx context.Context,
		sessionTrees []relayer.SessionTree,
	) (_ either.SessionTrees, skip bool) {
		if len(sessionTrees) == 0 {
			return either.Success(sessionTrees), false
		}

		// Map key is the supplier operator address.
		proofMsgs := make([]client.MsgSubmitProof, len(sessionTrees))
		for idx, session := range sessionTrees {
			proofMsgs[idx] = &prooftypes.MsgSubmitProof{
				SupplierOperatorAddress: session.GetSupplierOperatorAddress(),
				SessionHeader:           session.GetSessionHeader(),
				Proof:                   session.GetProofBz(),
			}
		}

		// Guard against empty sessionTrees to prevent index out of bounds errors
		if len(sessionTrees) == 0 {
			return either.Success(sessionTrees), false
		}

		// All session trees in the batch share the same sessionEndHeight, so we
		// can use the first one to calculate the proof window close height.
		//
		// TODO_REFACTOR(@red-0ne): Pass a richer type to the function instead of []SessionTrees to:
		// - Avoid making assumptions about shared properties
		// - Eliminate constant queries for sharedParams
		sessionEndHeight := sessionTrees[0].GetSessionHeader().GetSessionEndBlockHeight()
		sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
		if err != nil {
			failedSubmitProofSessionsCh <- sessionTrees
			rs.logger.Error().Err(err).Msg("❌️ Failed to retrieve shared network parameters. ❗Check node connectivity. ❗Rewards may not be secured and supplier may be slashed.")
			return either.Error[[]relayer.SessionTree](err), false
		}
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(sharedParams, sessionEndHeight)

		// Submit proofs for each supplier operator address in `sessionTrees`.
		if err := supplierClient.SubmitProofs(ctx, proofWindowCloseHeight, proofMsgs...); err != nil {
			failedSubmitProofSessionsCh <- sessionTrees
			rs.logger.Error().Err(err).Msg("❌ Failed to submit proofs to the network. ❗Check node connectivity and transaction fees. ❗Rewards may not be secured and supplier may be slashed.")
			return either.Error[[]relayer.SessionTree](err), false
		}

		rs.logger.Info().Msgf(
			"🎯 Successfully submitted %d proofs to the network - rewards secured!",
			len(sessionTrees),
		)

		rs.deleteSessionTrees(ctx, sessionTrees)
		return either.Success(sessionTrees), false
	}
}

// proveClaims generates the proofs corresponding to the given sessionTrees,
// then sends the successful and failed proofs to their respective channels.
func (rs *relayerSessionsManager) proveClaims(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
	// The hash of this block is used to determine which branch of the proof
	// should be generated for.
	proofPathSeedBlock client.Block,
) (successProofs []relayer.SessionTree, failedProofs []relayer.SessionTree) {
	logger := rs.logger.With("method", "proveClaims")
	logger.Info().Msgf("🔍 Analyzing %d session trees to determine proof requirements", len(sessionTrees))

	// sessionTreesWithProofRequired will accumulate all the sessionTrees that
	// will require a proof to be submitted.
	sessionTreesWithProofRequired := make([]relayer.SessionTree, 0)
	for _, sessionTree := range sessionTrees {
		isProofRequired, err := rs.isProofRequired(ctx, sessionTree, proofPathSeedBlock)

		// If an error is encountered while determining if a proof is required,
		// do not create the claim since the proof requirement is unknown.
		// WARNING: Creating a claim and not submitting a proof (if necessary) could lead to a stake burn!!
		if err != nil {
			failedProofs = append(failedProofs, sessionTree)
			logger.Error().Err(err).Msg("⚠️ Failed to determine if proof is required for session. ❗Check network connectivity")
			continue
		}

		// If a proof is required, add the session to the list of sessions that require a proof.
		if isProofRequired {
			sessionTreesWithProofRequired = append(sessionTreesWithProofRequired, sessionTree)
		} else {
			rs.deleteSessionTree(sessionTree)
		}
	}

	logger.Info().Msgf(
		"📊 Proof analysis complete: %d sessions require proofs, %d sessions skipped (no proof needed)",
		len(sessionTreesWithProofRequired), len(sessionTrees)-len(sessionTreesWithProofRequired),
	)

	// Separate the sessionTrees into those that failed to generate a proof
	// and those that succeeded, before returning each of them.
	for _, sessionTree := range sessionTreesWithProofRequired {
		// Generate the proof path for the sessionTree using the previously committed
		// sessionPathBlock hash.
		path := protocol.GetPathForProof(
			proofPathSeedBlock.Hash(),
			sessionTree.GetSessionHeader().GetSessionId(),
		)

		// If the proof cannot be generated, add the sessionTree to the failedProofs.
		if _, err := sessionTree.ProveClosest(path); err != nil {
			logger.Error().Err(err).Msg("⚠️ Failed to generate cryptographic proof for session. ❗Check session tree integrity and storage health.")

			failedProofs = append(failedProofs, sessionTree)
			continue
		}

		// If the proof was generated successfully, add the sessionTree to the
		// successProofs slice that will be sent to the proof submission step.
		successProofs = append(successProofs, sessionTree)
	}

	if len(successProofs) > 0 {
		logger.Info().Msgf(
			"✅ Successfully generated %d cryptographic proofs ready for submission",
			len(successProofs),
		)
	}

	if len(failedProofs) > 0 {
		logger.Warn().Msgf(
			"⚠️ Failed to generate proofs for %d sessions. ❗Check storage health and data integrity.",
			len(failedProofs),
		)
	}

	return successProofs, failedProofs
}

// isProofRequired determines whether a proof is required for the given session's
// claim based on the current proof module governance parameters.
// TODO_TECHDEBT: Refactor the method to be static and used both onchain and offchain.
// TODO_INVESTIGATE: Passing a polylog.Logger should allow for onchain/offchain
// usage of this function but it is currently raising a type error.
func (rs *relayerSessionsManager) isProofRequired(
	ctx context.Context,
	sessionTree relayer.SessionTree,
	// The hash of this block is used to determine whether the proof is required
	// w.r.t. the probabilistic features.
	proofRequirementSeedBlock client.Block,
) (isProofRequired bool, err error) {
	logger := rs.logger.With(
		"session_id", sessionTree.GetSessionHeader().GetSessionId(),
		"claim_root", fmt.Sprintf("%x", sessionTree.GetClaimRoot()),
		"supplier_operator_address", sessionTree.GetSupplierOperatorAddress(),
	)

	// Create the claim object and use its methods to determine if a proof is required.
	claim := claimFromSessionTree(sessionTree)

	proofParams, err := rs.proofQueryClient.GetParams(ctx)
	if err != nil {
		return false, err
	}

	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		return false, err
	}

	// Retrieving the relay mining difficulty for the service at hand
	serviceId := claim.GetSessionHeader().GetServiceId()
	relayMiningDifficulty, err := rs.serviceQueryClient.GetServiceRelayDifficulty(ctx, serviceId)
	if err != nil {
		return false, err
	}

	// The amount of uPOKT being claimed.
	claimedAmount, err := claim.GetClaimeduPOKT(*sharedParams, relayMiningDifficulty)
	if err != nil {
		return false, err
	}

	logger = logger.With(
		"claimed_amount_upokt", claimedAmount.Amount.Uint64(),
		"proof_requirement_threshold_upokt", proofParams.GetProofRequirementThreshold(),
	)

	// Require a proof if the claimed amount meets or exceeds the threshold.
	// TODO_MAINNET: This should be proportional to the supplier's stake as well.
	if claimedAmount.Amount.GTE(proofParams.GetProofRequirementThreshold().Amount) {
		logger.Info().Msg("💎 Claim value exceeds threshold - proof required to secure high-value rewards")

		return true, nil
	}

	proofRequirementSampleValue, err := claim.GetProofRequirementSampleValue(proofRequirementSeedBlock.Hash())
	if err != nil {
		return false, err
	}

	logger = logger.With(
		"proof_requirement_sample_value", proofRequirementSampleValue,
		"proof_request_probability", proofParams.GetProofRequestProbability(),
	)

	// Require a proof probabilistically based on the proof_request_probability param.
	// NB: A random value between 0 and 1 will be less than or equal to proof_request_probability
	// with probability equal to the proof_request_probability.
	if proofRequirementSampleValue <= proofParams.GetProofRequestProbability() {
		logger.Info().Msg("🎲 Random selection requires proof - contributing to network security through probabilistic verification")

		return true, nil
	}

	logger.Info().Msg("✅ Proof not required for this claim - proceeding without proof submission")
	return false, nil
}

// processProofsAsync handles the asynchronous processing of proofs to prevent blocking
// the observable pipeline. It waits for the appropriate block heights and then publishes
// ready sessions for proof submission without blocking request handling.
func (rs *relayerSessionsManager) processProofsAsync(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
	proofReadySessionsPublishCh chan<- []relayer.SessionTree,
	failedSubmitProofsSessionsPublishCh chan<- []relayer.SessionTree,
) {
	// FIX: Async Proofs Processing Implementation
	// This function runs in a separate goroutine to prevent blocking the main pipeline
	// while waiting for specific block heights required for proof submission.
	// This ensures new relay requests can continue to be processed while proofs
	// are being prepared in the background.

	logger := rs.logger.With("method", "processProofsAsync")

	// Perform the blocking wait operations in this goroutine
	sessionTreesWithProofs := rs.waitForEarliestSubmitProofsHeightAndGenerateProofs(
		ctx, sessionTrees, failedSubmitProofsSessionsPublishCh,
	)

	if sessionTreesWithProofs == nil {
		logger.Debug().Msg("No sessions with proofs after waiting for block height")
		return
	}

	// Check if context is still valid before proceeding
	if ctx.Err() != nil {
		logger.Warn().Err(ctx.Err()).Msg("Context canceled during async proof processing")
		return
	}

	logger.Info().Msgf(
		"✅ Async proof processing completed for %d sessions - publishing for proof submission",
		len(sessionTreesWithProofs),
	)

	// Publish the sessions that are ready to submit proofs
	// This notifies the proofs pipeline to proceed with actual proof submission
	select {
	case proofReadySessionsPublishCh <- sessionTreesWithProofs:
		logger.Debug().Msg("Successfully published sessions ready for proof submission")
	case <-ctx.Done():
		logger.Warn().Msg("Context canceled while publishing proof-ready sessions")
	}
}
