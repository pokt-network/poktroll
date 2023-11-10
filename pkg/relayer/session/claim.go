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

// createClaims maps over the sessionsToClaim observable. For each claim, it:
// 1. Calculates the earliest block height at which it is safe to CreateClaim
// 2. Waits for said block and creates the claim on-chain
// 3. Maps errors to a new observable and logs them
// 4. Returns an observable of the successfully claimed sessions
// It DOES NOT BLOCK as map operations run in their own goroutines.
func (rs *relayerSessionsManager) createClaims(ctx context.Context) observable.Observable[relayer.SessionTree] {
	// Map SessionsToClaim observable to a new observable of the same type which
	// is notified when the session is eligible to be claimed.
	// relayer.SessionTree ==> relayer.SessionTree
	sessionsWithOpenClaimWindowObs := channel.Map(
		ctx, rs.sessionsToClaim,
		rs.mapWaitForEarliestCreateClaimHeight,
	)

	failedCreateClaimSessionsObs, failedCreateClaimSessionsPublishCh :=
		channel.NewObservable[relayer.SessionTree]()

	// Map sessionsWithOpenClaimWindow to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// claim has been created or an error has been encountered, respectively.
	eitherClaimedSessionsObs := channel.Map(
		ctx, sessionsWithOpenClaimWindow,
		rs.newMapClaimSessionFn(failedCreateClaimSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed create claim sessions to some retry mechanism.
	_ = failedCreateClaimSessions
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherClaimedSessions))

	// Map eitherClaimedSessions to a new observable of relayer.SessionTree which
	// is notified when the corresponding claim creation succeeded.
	return filter.EitherSuccess(ctx, eitherClaimedSessions)
}

// mapWaitForEarliestCreateClaimHeight is intended to be used as a MapFn. It
// calculates and waits for the earliest block height, allowed by the protocol,
// at which a claim can be created for the given session, then emits the session
// **at that moment**.
func (rs *relayerSessionsManager) mapWaitForEarliestCreateClaimHeight(
	ctx context.Context,
	session relayer.SessionTree,
) (_ relayer.SessionTree, skip bool) {
	rs.waitForEarliestCreateClaimHeight(
		ctx, session.GetSessionHeader().GetSessionEndBlockHeight(),
	)
	return session, false
}

// waitForEarliestCreateClaimHeight calculates and waits for (blocking until) the
// earliest block height, allowed by the protocol, at which a claim can be created
// for a session with the given sessionEndHeight. It is calculated relative to
// sessionEndHeight using on-chain governance parameters and randomized input.
// It IS A BLOCKING function.
func (rs *relayerSessionsManager) waitForEarliestCreateClaimHeight(
	ctx context.Context,
	sessionEndHeight int64,
) {
	// TODO_TECHDEBT: refactor this logic to a shared package.

	createClaimWindowStartHeight := sessionEndHeight
	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// + claimproofparams.GovCreateClaimWindowStartHeightOffset

	// we wait for createClaimWindowStartHeight to be received before proceeding since we need its hash
	// to know where this servicer's claim submission window starts.
	log.Printf("waiting & blocking for global earliest claim submission createClaimWindowStartBlock height: %d", createClaimWindowStartHeight)
	createClaimWindowStartBlock := rs.waitForBlock(ctx, createClaimWindowStartHeight)

	log.Printf("received earliest claim submission createClaimWindowStartBlock height: %d, use its hash to have a random submission for the servicer", createClaimWindowStartBlock.Height())

	earliestCreateClaimHeight :=
		protocol.GetEarliestCreateClaimHeight(createClaimWindowStartBlock)

	log.Printf("earliest claim submission createClaimWindowStartBlock height for this supplier: %d", earliestCreateClaimHeight)
	_ = rs.waitForBlock(ctx, earliestCreateClaimHeight)
}

// newMapClaimSessionFn returns a new MapFn that creates a claim for the given
// session. Any session which encouters an error while creating a claim is sent
// on the failedCreateClaimSessions channel.
func (rs *relayerSessionsManager) newMapClaimSessionFn(
	failedCreateClaimSessionsPublishCh chan<- relayer.SessionTree,
) channel.MapFn[relayer.SessionTree, either.SessionTree] {
	return func(
		ctx context.Context,
		session relayer.SessionTree,
	) (_ either.SessionTree, skip bool) {
		// this session should no longer be updated
		claimRoot, err := session.Flush()
		if err != nil {
			return either.Error[relayer.SessionTree](err), false
		}

		sessionHeader := session.GetSessionHeader()
		if err := rs.supplierClient.CreateClaim(ctx, *sessionHeader, claimRoot); err != nil {
			failedCreateClaimSessionsPublishCh <- session
			return either.Error[relayer.SessionTree](err), false
		}

		return either.Success(session), false
	}
}
