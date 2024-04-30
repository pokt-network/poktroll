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
	claimedSessionsObs observable.Observable[*relayer.SessionTreeBatch],
) {
	// Map claimedSessionsObs to a new observable of the same type which is notified
	// when the sessions in the batch are eligible to be proven.
	sessionsWithOpenProofWindowObs := channel.Map(
		ctx, claimedSessionsObs,
		rs.mapWaitForEarliestSubmitProofsHeight,
	)

	failedSubmitProofsSessionsObs, failedSubmitProofsSessionsPublishCh :=
		channel.NewObservable[*relayer.SessionTreeBatch]()

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
	ctx context.Context,
	sessionTreeBatch *relayer.SessionTreeBatch,
) (_ *relayer.SessionTreeBatch, skip bool) {
	rs.waitForEarliestSubmitProofsHeight(
		ctx, sessionTreeBatch.SessionsEndBlockHeight,
	)
	return sessionTreeBatch, false
}

// waitForEarliestSubmitProofsHeight calculates and waits for (blocking until) the
// earliest block height, allowed by the protocol, at which proofs can be submitted
// for a session number which where claimed at createClaimHeight. It is calculated relative
// to createClaimHeight using on-chain governance parameters and randomized input.
// It IS A BLOCKING function.
func (rs *relayerSessionsManager) waitForEarliestSubmitProofsHeight(
	ctx context.Context,
	createClaimHeight int64,
) {
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

	earliestSubmitProofsHeight := protocol.GetEarliestSubmitProofsHeight(ctx, submitProofsWindowStartBlock)
	_ = rs.waitForBlock(ctx, earliestSubmitProofsHeight)
}

// newMapProveSessionsFn returns a new MapFn that submits proofs on the given
// session batch. Any session which encounters errors while submitting a proof is sent
// on the failedSubmitProofSessions channel.
func (rs *relayerSessionsManager) newMapProveSessionsFn(
	failedSubmitProofSessionsCh chan<- *relayer.SessionTreeBatch,
) channel.MapFn[*relayer.SessionTreeBatch, either.SessionTreeBatch] {
	return func(
		ctx context.Context,
		sessionTreeBatch *relayer.SessionTreeBatch,
	) (_ either.SessionTreeBatch, skip bool) {
		// TODO_BLOCKER: The block that'll be used as a source of entropy for which
		// branch(es) to prove should be deterministic and use on-chain governance params
		// rather than latest.
		pathBlockHeight := sessionTreeBatch.SessionsEndBlockHeight +
			sessionkeeper.GetSessionGracePeriodBlockCount()
		pathBlock, err := rs.blockQueryClient.Block(ctx, &pathBlockHeight)
		if err != nil {
			return either.Error[*relayer.SessionTreeBatch](err), false
		}

		sessionProofs := []*relayer.SessionProof{}
		for _, sessionTree := range sessionTreeBatch.SessionTrees {
			path := proofkeeper.GetPathForProof(
				pathBlock.BlockID.Hash,
				sessionTree.GetSessionHeader().GetSessionId(),
			)
			proof, err := sessionTree.ProveClosest(path)
			if err != nil {
				return either.Error[*relayer.SessionTreeBatch](err), false
			}

			proofBz, err := proof.Marshal()
			if err != nil {
				return either.Error[*relayer.SessionTreeBatch](err), false
			}

			sessionProofs = append(sessionProofs, &relayer.SessionProof{
				ProofBz:       proofBz,
				SessionHeader: sessionTree.GetSessionHeader(),
			})
		}

		rs.logger.Info().
			Int64("session_start_height", pathBlock.Block.Height).
			Msg("submitting proof")

		// SubmitProof ensures on-chain proof inclusion so we can safely prune the tree.
		if err := rs.supplierClient.SubmitProofs(
			ctx,
			sessionProofs,
		); err != nil {
			failedSubmitProofSessionsCh <- sessionTreeBatch
			return either.Error[*relayer.SessionTreeBatch](err), false
		}

		return either.Success(sessionTreeBatch), false
	}
}
