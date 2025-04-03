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
// 1. Calculates the earliest block height at which to submit proofs
// 2. Waits for said height and submits the proofs onchain
// 3. Maps errors to a new observable and logs them
// It DOES NOT BLOCK as map operations run in their own goroutines.
func (rs *relayerSessionsManager) submitProofs(
	ctx context.Context,
	supplierClient client.SupplierClient,
	claimedSessionsObs observable.Observable[[]relayer.SessionTree],
) {
	failedSubmitProofsSessionsObs, failedSubmitProofsSessionsPublishCh :=
		channel.NewObservable[[]relayer.SessionTree]()

	// Map claimedSessionsObs to a new observable of the same type which is notified
	// when the sessions in the batch are eligible to be proven.
	sessionsWithOpenProofWindowObs := channel.Map(
		ctx, claimedSessionsObs,
		rs.mapWaitForEarliestSubmitProofsHeight(failedSubmitProofsSessionsPublishCh),
	)

	// Map sessionsWithOpenProofWindow to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// proof has been submitted or an error has been encountered, respectively.
	eitherProvenSessionsObs := channel.Map(
		ctx, sessionsWithOpenProofWindowObs,
		rs.newMapProveSessionsFn(supplierClient, failedSubmitProofsSessionsPublishCh),
	)

	logging.LogErrors(ctx, filter.EitherError(ctx, eitherProvenSessionsObs))

	// Delete expired session trees so they don't get proven again.
	channel.ForEach(
		ctx, failedSubmitProofsSessionsObs,
		rs.deleteExpiredSessionTreesFn(sharedtypes.GetProofWindowCloseHeight),
	)
}

// mapWaitForEarliestSubmitProofsHeight is intended to be used as a MapFn. It
// calculates and waits for the earliest block height, allowed by the protocol,
// at which proofs can be submitted for the given session number, then emits the session
// **at that moment**.
func (rs *relayerSessionsManager) mapWaitForEarliestSubmitProofsHeight(
	failSubmitProofsSessionsCh chan<- []relayer.SessionTree,
) channel.MapFn[[]relayer.SessionTree, []relayer.SessionTree] {
	return func(
		ctx context.Context,
		sessionTrees []relayer.SessionTree,
	) (_ []relayer.SessionTree, skip bool) {
		return rs.waitForEarliestSubmitProofsHeightAndGenerateProofs(
			ctx, sessionTrees, failSubmitProofsSessionsCh,
		), false
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
	// Given the sessionTrees are grouped by their sessionEndHeight, we can use the
	// first one from the group to calculate the earliest height for proof submission.
	sessionEndHeight := sessionTrees[0].GetSessionHeader().GetSessionEndBlockHeight()

	logger := rs.logger.With("session_end_height", sessionEndHeight)

	// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
	// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
	// to get the most recently (asynchronously) observed (and cached) value.
	// TODO_MAINNET(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
	// we should be using the value that the params had for the session which includes queryHeight.
	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get shared params")
		failedSubmitProofsSessionsCh <- sessionTrees
		return nil
	}

	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, sessionEndHeight)

	// we wait for proofWindowOpenHeight to be received before proceeding since we need
	// its hash to seed the pseudo-random number generator for the proof submission
	// distribution (i.e. earliestSupplierProofCommitHeight).
	logger = logger.With("proof_window_open_height", proofWindowOpenHeight)
	logger.Info().Msg("waiting & blocking until the proof window open height")

	proofsWindowOpenBlock := rs.waitForBlock(ctx, proofWindowOpenHeight)
	// TODO_MAINNET: If a relayminer is cold-started with persisted but unproven ("late")
	// sessions, the proofsWindowOpenBlock will never be observed. Where a "late" session
	// is one whic is unclaimed and whose earliest claim commit height has already elapsed.
	//
	// In this case, we should
	// use a block query client to populate the block client replay observable at the time
	// of block client construction. This check and failure branch can be removed once this
	// is implemented.
	if proofsWindowOpenBlock == nil {
		logger.Warn().Msg("failed to observe earliest proof commit height offset seed block height")
		failedSubmitProofsSessionsCh <- sessionTrees
		return nil
	}

	// Get the earliest proof commit height for this supplier.
	supplierOperatorAddr := sessionTrees[0].GetSupplierOperatorAddress()
	earliestSupplierProofsCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		sessionEndHeight,
		proofsWindowOpenBlock.Hash(),
		supplierOperatorAddr,
	)

	logger = logger.With("earliest_supplier_proof_commit_height", earliestSupplierProofsCommitHeight)
	logger.Info().Msg("waiting & blocking for proof path seed block height")

	// earliestSupplierProofsCommitHeight - 1 is the block that will have its hash
	// used as the source of entropy for all the session trees in that batch,
	// waiting for it to be received before proceeding.
	proofPathSeedBlockHeight := earliestSupplierProofsCommitHeight - 1
	proofPathSeedBlock := rs.waitForBlock(ctx, proofPathSeedBlockHeight)

	logger = logger.With("proof_path_seed_block", fmt.Sprintf("%x", proofPathSeedBlock.Hash()))
	logger.Info().Msg("observed proof path seed block height")

	successProofs, failedProofs := rs.proveClaims(ctx, sessionTrees, proofPathSeedBlock)
	failedSubmitProofsSessionsCh <- failedProofs

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
			rs.logger.Error().Err(err).Msg("failed to get shared params")
			return either.Error[[]relayer.SessionTree](err), false
		}
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(sharedParams, sessionEndHeight)

		// Submit proofs for each supplier operator address in `sessionTrees`.
		if err := supplierClient.SubmitProofs(ctx, proofWindowCloseHeight, proofMsgs...); err != nil {
			failedSubmitProofSessionsCh <- sessionTrees
			rs.logger.Error().Err(err).Msg("failed to submit proofs")
			return either.Error[[]relayer.SessionTree](err), false
		}

		for _, sessionTree := range sessionTrees {
			rs.removeFromRelayerSessions(sessionTree)
			if err := sessionTree.Delete(); err != nil {
				// Do not fail the entire operation if a session tree cannot be deleted
				// as this does not affect the C&P lifecycle.
				rs.logger.Error().Err(err).Msg("failed to delete session tree")
			}
		}

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
			logger.Error().Err(err).Msg("failed to determine if proof is required, skipping claim creation")
			continue
		}

		// If a proof is required, add the session to the list of sessions that require a proof.
		if isProofRequired {
			sessionTreesWithProofRequired = append(sessionTreesWithProofRequired, sessionTree)
		} else {
			rs.removeFromRelayerSessions(sessionTree)
			if err := sessionTree.Delete(); err != nil {
				// Do not fail the entire operation if a session tree cannot be deleted
				// as this does not affect the C&P lifecycle.
				logger.Error().Err(err).Msg("failed to delete session tree")
			}
		}
	}

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
			logger.Error().Err(err).Msg("failed to generate proof")

			failedProofs = append(failedProofs, sessionTree)
			continue
		}

		// If the proof was generated successfully, add the sessionTree to the
		// successProofs slice that will be sent to the proof submission step.
		successProofs = append(successProofs, sessionTree)
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
		logger.Info().Msg("compute units is above threshold, claim requires proof")

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
		logger.Info().Msg("claim hash seed is below proof request probability, claim requires proof")

		return true, nil
	}

	logger.Info().Msg("claim does not require proof")
	return false, nil
}

// claimFromSessionTree creates a claim object from the given SessionTree.
func claimFromSessionTree(sessionTree relayer.SessionTree) prooftypes.Claim {
	return prooftypes.Claim{
		SupplierOperatorAddress: sessionTree.GetSupplierOperatorAddress(),
		SessionHeader:           sessionTree.GetSessionHeader(),
		RootHash:                sessionTree.GetClaimRoot(),
	}
}
