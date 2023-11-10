package session

import (
	"context"
	"log"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
)

// submitProofs maps over the given claimedSessions observable. For each session,
// it calculates and waits for the earliest block height at which it is safe to
// submit a proof and does so. It then maps any errors to a new observable which
// are subsequently logged. It does not block as map operations run in their own
// goroutines.
func (rs *relayerSessionsManager) submitProofs(
	ctx context.Context,
	claimedSessions observable.Observable[relayer.SessionTree],
) {
	// Map claimedSessions to a new observable of the same type which is notified
	// when the session is eligible to be proven.
	sessionsWithOpenProofWindow := channel.Map(
		ctx, claimedSessions,
		rs.mapWaitForEarliestSubmitProofHeight,
	)

	failedSubmitProofSessions, failedSubmitProveSessionsPublishCh :=
		channel.NewObservable[relayer.SessionTree]()

	// Map sessionsWithOpenProofWindow to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// proof has been submitted or an error has been encountered, respectively.
	eitherProvenSessions := channel.Map(
		ctx, sessionsWithOpenProofWindow,
		rs.newMapProveSessionFn(failedSubmitProveSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed submit proof sessions to some retry mechanism.
	_ = failedSubmitProofSessions
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherProvenSessions))
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
func (rs *relayerSessionsManager) waitForEarliestSubmitProofHeight(
	ctx context.Context,
	createClaimHeight int64,
) {
	submitProofWindowStartHeight := createClaimHeight
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// + claimproofparams.GovSubmitProofWindowStartHeightOffset

	// we wait for submitProofWindowStartHeight to be received before proceeding since we need its hash
	log.Printf("waiting for global earliest proof submission submitProofWindowStartBlock height: %d", submitProofWindowStartHeight)
	submitProofWindowStartBlock := rs.waitForBlock(ctx, submitProofWindowStartHeight)

	earliestSubmitProofHeight := protocol.GetEarliestSubmitProofHeight(submitProofWindowStartBlock)
	_ = rs.waitForBlock(ctx, earliestSubmitProofHeight)
}

// newMapProveSessionFn returns a new MapFn that submits a proof for the given
// session. Any session which encouters errors while submitting a proof is sent
// on the failedSubmitProofSessions channel.
func (rs *relayerSessionsManager) newMapProveSessionFn(
	failedSubmitProofSessions chan<- relayer.SessionTree,
) channel.MapFn[relayer.SessionTree, either.SessionTree] {
	return func(
		ctx context.Context,
		session relayer.SessionTree,
	) (_ either.SessionTree, skip bool) {
		latestBlock := rs.blockClient.LatestBlock(ctx)
		proof, err := session.ProveClosest(latestBlock.Hash())
		if err != nil {
			return either.Error[relayer.SessionTree](err), false
		}

		log.Printf("currentBlock: %d, submitting proof", latestBlock.Height()+1)
		// SubmitProof ensures on-chain proof inclusion so we can safely prune the tree.
		if err := rs.supplierClient.SubmitProof(
			ctx,
			*session.GetSessionHeader(),
			proof,
		); err != nil {
			failedSubmitProofSessions <- session
			return either.Error[relayer.SessionTree](err), false
		}

		return either.Success(session), false
	}
}
