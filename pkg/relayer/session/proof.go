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
	"github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/shared"
)

// submitProofs maps over the given claimedSessions observable.
// For each session batch, it:
// 1. Calculates the earliest block height at which to submit proofs
// 2. Waits for said height and submits the proofs on-chain
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

	// TODO_TECHDEBT: pass failed submit proof sessions to some retry mechanism.
	_ = failedSubmitProofsSessionsObs
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherProvenSessionsObs))

	// Delete expired session trees so they don't get proven again.
	channel.ForEach(
		ctx, failedSubmitProofsSessionsObs,
		rs.deleteExpiredSessionTreesFn(shared.GetProofWindowCloseHeight),
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
// It is calculated relative to createClaimHeight using on-chain governance parameters
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

	// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
	// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
	// to get the most recently (asynchronously) observed (and cached) value.
	// TODO_BLOCKER(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
	// we should be using the value that the params had for the session which includes queryHeight.
	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get shared params")
		failedSubmitProofsSessionsCh <- sessionTrees
		return nil
	}

	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(sharedParams, sessionEndHeight)

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
	supplierAddr := sessionTrees[0].GetSupplierAddress().String()
	earliestSupplierProofsCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		sessionEndHeight,
		proofsWindowOpenBlock.Hash(),
		supplierAddr,
	)

	logger = logger.With("earliest_supplier_proof_commit_height", earliestSupplierProofsCommitHeight)
	logger.Info().Msg("waiting & blocking for proof path seed block height")

	// proofWindowOpenHeight - 1 is the block that will have its hash used as the
	// source of entropy for all the session trees in that batch, waiting for it to
	// be received before proceeding.
	proofPathSeedBlockHeight := earliestSupplierProofsCommitHeight - 1
	proofPathSeedBlock := rs.waitForBlock(ctx, proofPathSeedBlockHeight)

	logger = logger.With("proof_path_bock_hash", fmt.Sprintf("%x", proofPathSeedBlock.Hash()))
	logger.Info().Msg("observed proof path seed block height")

	// Generate proofs for all sessionTrees concurrently while waiting for the
	// earliest submitProofsHeight (pseudorandom submission distribution) to be reached.
	// Use a channel to block until all proofs for the sessionTrees have been generated.
	proofsGeneratedCh := make(chan []relayer.SessionTree)
	defer close(proofsGeneratedCh)
	go rs.goProveClaims(
		ctx,
		sessionTrees,
		proofPathSeedBlock,
		proofsGeneratedCh,
		failedSubmitProofsSessionsCh,
	)

	logger.Info().Msg("waiting & blocking for earliest supplier proof commit height")

	// Wait for the earliestSupplierProofsCommitHeight to be reached before proceeding.
	_ = rs.waitForBlock(ctx, earliestSupplierProofsCommitHeight)

	logger.Info().Msg("observed earliest supplier proof commit height")

	// Once the earliest submitProofsHeight has been reached, and all proofs have
	// been generated, return the sessionTrees that have been successfully proven
	// to be submitted on-chain.
	return <-proofsGeneratedCh
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

		// Map key is the supplier address.
		proofMsgs := make([]client.MsgSubmitProof, 0)
		for _, session := range sessionTrees {
			proofMsgs = append(proofMsgs, &types.MsgSubmitProof{
				Proof:           session.GetProofBz(),
				SessionHeader:   session.GetSessionHeader(),
				SupplierAddress: session.GetSupplierAddress().String(),
			})
		}

		// Submit proofs for each supplier address in `sessionTrees`.
		if err := supplierClient.SubmitProofs(ctx, proofMsgs...); err != nil {
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

// goProveClaims generates the proofs corresponding to the given sessionTrees,
// then sends the successful and failed proofs to their respective channels.
// This function MUST be run as a goroutine.
func (rs *relayerSessionsManager) goProveClaims(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
	sessionPathBlock client.Block,
	proofsGeneratedCh chan<- []relayer.SessionTree,
	failSubmitProofsSessionsCh chan<- []relayer.SessionTree,
) {
	logger := rs.logger.With("method", "goProveClaims")

	// Separate the sessionTrees into those that failed to generate a proof
	// and those that succeeded, then send them on their respective channels.
	failedProofs := []relayer.SessionTree{}
	successProofs := []relayer.SessionTree{}
	for _, sessionTree := range sessionTrees {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// Generate the proof path for the sessionTree using the previously committed
		// sessionPathBlock hash.
		path := protocol.GetPathForProof(
			sessionPathBlock.Hash(),
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

	failSubmitProofsSessionsCh <- failedProofs
	proofsGeneratedCh <- successProofs
}
