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
// For each session, it:
// 1. Calculates the earliest block height at which to submit a proof
// 2. Waits for said height and submits the proof on-chain
// 3. Maps errors to a new observable and logs them
// It DOES NOT BLOCKas map operations run in their own goroutines.
func (rs *relayerSessionsManager) submitProofs(
	ctx context.Context,
	claimedSessionsObs observable.Observable[relayer.SessionTree],
) {
	// Map claimedSessionsObs to a new observable of the same type which is notified
	// when the session is eligible to be proven.
	sessionsWithOpenProofWindowObs := channel.Map(
		ctx, claimedSessionsObs,
		rs.mapWaitForEarliestSubmitProofHeight,
	)

	failedSubmitProofSessionsObs, failedSubmitProofSessionsPublishCh := channel.NewObservable[relayer.SessionTree]()

	// Map sessionsWithOpenProofWindow to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// proof has been submitted or an error has been encountered, respectively.
	eitherProvenSessionsObs := channel.Map(
		ctx, sessionsWithOpenProofWindowObs,
		rs.newMapProveSessionFn(failedSubmitProofSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed submit proof sessions to some retry mechanism.
	_ = failedSubmitProofSessionsObs
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherProvenSessionsObs))
}

// mapWaitForEarliestSubmitProofHeight is intended to be used as a MapFn. It
// calculates and waits for the earliest block height, allowed by the protocol,
// at which a proof can be submitted for the given session, then emits the session
// **at that moment**.
func (rs *relayerSessionsManager) mapWaitForEarliestSubmitProofHeight(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	rs.waitForEarliestSubmitProofHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)
	return session, false
}

// waitForEarliestSubmitProofHeight calculates and waits for (blocking until) the
// earliest block height, allowed by the protocol, at which a proof can be submitted
// for a session which was claimed at createClaimHeight. It is calculated relative
// to createClaimHeight using on-chain governance parameters and randomized input.
// It IS A BLOCKING function.
func (rs *relayerSessionsManager) waitForEarliestSubmitProofHeight(
	ctx context.Context,
	createClaimHeight int64,
) {
	// TODO_TECHDEBT(@red-0ne): Centralize the business logic that involves taking
	// into account the heights, windows and grace periods into helper functions.
	submitProofWindowStartHeight := createClaimHeight + sessionkeeper.GetSessionGracePeriodBlockCount()
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// + claimproofparams.GovSubmitProofWindowStartHeightOffset

	// we wait for submitProofWindowStartHeight to be received before proceeding since we need its hash
	rs.logger.Info().
		Int64("submitProofWindowStartHeight", submitProofWindowStartHeight).
		Msg("waiting & blocking for global earliest proof submission height")
	submitProofWindowStartBlock := rs.waitForBlock(ctx, submitProofWindowStartHeight)

	earliestSubmitProofHeight := protocol.GetEarliestSubmitProofHeight(ctx, submitProofWindowStartBlock)
	_ = rs.waitForBlock(ctx, earliestSubmitProofHeight)
}

// newMapProveSessionFn returns a new MapFn that submits a proof for the given
// session. Any session which encouters errors while submitting a proof is sent
// on the failedSubmitProofSessions channel.
func (rs *relayerSessionsManager) newMapProveSessionFn(
	failedSubmitProofSessionsCh chan<- relayer.SessionTree,
) channel.MapFn[relayer.SessionTree, either.SessionTree] {
	return func(
		ctx context.Context,
		session relayer.SessionTree,
	) (_ either.SessionTree, skip bool) {
		// TODO_BLOCKER: The block that'll be used as a source of entropy for which
		// branch(es) to prove should be deterministic and use on-chain governance params
		// rather than latest.
		latestBlock := rs.blockClient.LastNBlocks(ctx, 1)[0]
		// TODO_BLOCKER(@red-0ne, @Olshansk): Update the path given to `ProveClosest`
		// from `BlockHash` to `Foo(BlockHash, SessionId)`
		path := proofkeeper.GetPathForProof(latestBlock.Hash(), session.GetSessionHeader().GetSessionId())
		proof, err := session.ProveClosest(path)
		if err != nil {
			return either.Error[relayer.SessionTree](err), false
		}

		rs.logger.Info().
			Int64("currentBlockHeight", latestBlock.Height()+1).
			Msg("submitting proof")

		// SubmitProof ensures on-chain proof inclusion so we can safely prune the tree.
		if err := rs.supplierClient.SubmitProof(
			ctx,
			*session.GetSessionHeader(),
			proof,
		); err != nil {
			failedSubmitProofSessionsCh <- session
			return either.Error[relayer.SessionTree](err), false
		}

		return either.Success(session), false
	}
}
