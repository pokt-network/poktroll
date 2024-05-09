package session

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	proofkeeper "github.com/pokt-network/poktroll/x/proof/keeper"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// submitProofs maps over the given claimedSessions observable.
// For each session batch, it:
// 1. Calculates the earliest block height at which to submit proofs
// 2. Waits for said height and submits the proofs on-chain
// 3. Maps errors to a new observable and logs them
// It DOES NOT BLOCKas map operations run in their own goroutines.
func (rs *relayerSessionsManager) submitProofs(
	ctx context.Context,
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
		rs.newMapProveSessionsFn(failedSubmitProofsSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed submit proof sessions to some retry mechanism.
	_ = failedSubmitProofsSessionsObs
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherProvenSessionsObs))
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
		return rs.waitForEarliestSubmitProofsHeightAndGenerateProof(
			ctx, sessionTrees, failSubmitProofsSessionsCh,
		), false
	}
}

// waitForEarliestSubmitProofsHeightAndGenerateProof calculates and waits for
// (blocking until) the earliest block height, allowed by the protocol, at which
// proofs can be submitted for a session number which where claimed at createClaimHeight.
// It is calculated relative to createClaimHeight using on-chain governance parameters
// and randomized input.
func (rs *relayerSessionsManager) waitForEarliestSubmitProofsHeightAndGenerateProof(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
	failSubmitProofsSessionsCh chan<- []relayer.SessionTree,
) []relayer.SessionTree {
	createClaimHeight := sessionTrees[0].GetSessionHeader().GetSessionEndBlockHeight()
	// TODO_TECHDEBT(@red-0ne): Centralize the business logic that involves taking
	// into account the heights, windows and grace periods into helper functions.
	submitProofsWindowStartHeight := createClaimHeight + sessionkeeper.GetSessionGracePeriodBlockCount()
	// TODO_BLOCKER: query the on-chain governance parameter once available.
	// + claimproofparams.GovSubmitProofWindowStartHeightOffset

	// we wait for submitProofsWindowStartHeight to be received before proceeding since we need its hash
	rs.logger.Info().
		Int64("submitProofsWindowStartHeight", submitProofsWindowStartHeight).
		Msg("waiting & blocking for global earliest proof submission height")
	submitProofsWindowStartBlock := rs.waitForBlock(ctx, submitProofsWindowStartHeight)

	proveDone := make(chan []relayer.SessionTree)
	defer close(proveDone)
	go func() {
		failedProofs := []relayer.SessionTree{}
		successProofs := []relayer.SessionTree{}
		for _, sessionTree := range sessionTrees {
			path := proofkeeper.GetPathForProof(
				submitProofsWindowStartBlock.Hash(),
				sessionTree.GetSessionHeader().GetSessionId(),
			)

			if _, err := sessionTree.ProveClosest(path); err != nil {
				failedProofs = append(failedProofs, sessionTree)
				continue
			}

			successProofs = append(successProofs, sessionTree)
		}
		failSubmitProofsSessionsCh <- failedProofs
		proveDone <- successProofs
	}()

	earliestSubmitProofsHeight := protocol.GetEarliestSubmitProofsHeight(ctx, submitProofsWindowStartBlock)
	_ = rs.waitForBlock(ctx, earliestSubmitProofsHeight)

	return <-proveDone
}

// newMapProveSessionsFn returns a new MapFn that submits proofs on the given
// session batch. Any session which encounters errors while submitting a proof is sent
// on the failedSubmitProofSessions channel.
func (rs *relayerSessionsManager) newMapProveSessionsFn(
	failedSubmitProofSessionsCh chan<- []relayer.SessionTree,
) channel.MapFn[[]relayer.SessionTree, either.SessionTrees] {
	return func(
		ctx context.Context,
		sessionTrees []relayer.SessionTree,
	) (_ either.SessionTrees, skip bool) {
		rs.pendingTxMu.Lock()
		defer rs.pendingTxMu.Unlock()

		if len(sessionTrees) == 0 {
			return either.Success(sessionTrees), false
		}

		sessionProofs := []*relayer.SessionProof{}
		for _, sessionTree := range sessionTrees {
			sessionProofs = append(sessionProofs, &relayer.SessionProof{
				ProofBz:       sessionTree.GetProofBz(),
				SessionHeader: sessionTree.GetSessionHeader(),
			})
		}

		sessionStartHeight := sessionTrees[0].GetSessionHeader().GetSessionStartBlockHeight()
		rs.logger.Info().
			Int64("session_start_height", sessionStartHeight).
			Msg("submitting proofs")

		// SubmitProof ensures on-chain proof inclusion so we can safely prune the tree.
		if err := rs.supplierClient.SubmitProofs(ctx, sessionProofs); err != nil {
			failedSubmitProofSessionsCh <- sessionTrees
			return either.Error[[]relayer.SessionTree](err), false
		}

		for _, sessionTree := range sessionTrees {
			// Prune the session tree after the proof has been submitted.
			if err := sessionTree.Delete(); err != nil {
				return either.Error[[]relayer.SessionTree](err), false
			}
		}

		return either.Success(sessionTrees), false
	}
}
