package session

import (
	"context"
	"fmt"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// createClaims maps over the sessionsToClaimObs observable. For each claim batch, it:
// 1. Calculates the earliest block height at which it is safe to CreateClaims
// 2. Waits for said block and creates the claims on-chain
// 3. Maps errors to a new observable and logs them
// 4. Returns an observable of the successfully claimed sessions
// It DOES NOT BLOCK as map operations run in their own goroutines.
func (rs *relayerSessionsManager) createClaims(
	ctx context.Context,
) observable.Observable[*relayer.SessionTreeBatch] {
	// Map sessionsToClaimObs to a new observable of the same type which is notified
	// when the session is eligible to be claimed.
	sessionsWithOpenClaimWindowObs := channel.Map(
		ctx, rs.sessionsToClaimsObs,
		rs.mapWaitForEarliestCreateClaimsHeight,
	)

	failedCreateClaimSessionsObs, failedCreateClaimSessionsPublishCh :=
		channel.NewObservable[*relayer.SessionTreeBatch]()

	// Map sessionsWithOpenClaimWindowObs to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// number claims have been created or an error has been encountered, respectively.
	eitherClaimedSessionsObs := channel.Map(
		ctx, sessionsWithOpenClaimWindowObs,
		rs.newMapClaimSessionsFn(failedCreateClaimSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed create claim sessions to some retry mechanism.
	_ = failedCreateClaimSessionsObs
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherClaimedSessionsObs))

	// Map eitherClaimedSessions to a new observable of relayer.SessionTreeBatch
	// which is notified when the corresponding claims creation succeeded.
	return filter.EitherSuccess(ctx, eitherClaimedSessionsObs)
}

// mapWaitForEarliestCreateClaimsHeight is intended to be used as a MapFn. It
// calculates and waits for the earliest block height, allowed by the protocol,
// at which claims can be created for the given session number, then emits the
// session **at that moment**.
func (rs *relayerSessionsManager) mapWaitForEarliestCreateClaimsHeight(
	ctx context.Context,
	sessionTreeBatch *relayer.SessionTreeBatch,
) (_ *relayer.SessionTreeBatch, skip bool) {
	rs.waitForEarliestCreateClaimsHeight(
		ctx, sessionTreeBatch.SessionsEndBlockHeight,
	)
	return sessionTreeBatch, false
}

// waitForEarliestCreateClaimsHeight calculates and waits for (blocking until) the
// earliest block height, allowed by the protocol, at which claims can be created
// for a session number with the given sessionEndHeight. It is calculated relative
// to sessionEndHeight using on-chain governance parameters and randomized input.
// It IS A BLOCKING function.
func (rs *relayerSessionsManager) waitForEarliestCreateClaimsHeight(
	ctx context.Context,
	sessionEndHeight int64,
) {
	logger := polylog.Ctx(ctx)

	// TODO_TECHDEBT(@red-0ne): Centralize the business logic that involves taking
	// into account the heights, windows and grace periods into helper functions.
	createClaimsWindowStartHeight := sessionEndHeight + sessionkeeper.GetSessionGracePeriodBlockCount()

	// TODO_BLOCKER: query the on-chain governance parameter once available.
	// + claimproofparams.GovCreateClaimWindowStartHeightOffset

	// we wait for createClaimsWindowStartHeight to be received before proceeding since we need its hash
	// to know where this servicer's claim submission window starts.
	logger.Info().
		Int64("create_claim_window_start_height", createClaimsWindowStartHeight).
		Msg("waiting & blocking for global earliest claim submission height")
	createClaimsWindowStartBlock := rs.waitForBlock(ctx, createClaimsWindowStartHeight)

	logger.Info().
		Int64("create_claim_window_start_height", createClaimsWindowStartBlock.Height()).
		Str("hash", fmt.Sprintf("%x", createClaimsWindowStartBlock.Hash())).
		Msg("received global earliest claim submission height")

	earliestCreateClaimsHeight := protocol.GetEarliestCreateClaimsHeight(
		ctx,
		createClaimsWindowStartBlock,
	)

	logger.Info().
		Int64("earliest_create_claim_height", earliestCreateClaimsHeight).
		Str("hash", fmt.Sprintf("%x", createClaimsWindowStartBlock.Hash())).
		Msg("waiting & blocking for earliest claim creation height for this supplier")

	_ = rs.waitForBlock(ctx, earliestCreateClaimsHeight)
}

// newMapClaimSessionsFn returns a new MapFn that creates claims for the given
// session number. Any session which encounters an error while creating a claim
// is sent on the failedCreateClaimSessions channel.
func (rs *relayerSessionsManager) newMapClaimSessionsFn(
	failedCreateClaimsSessionsPublishCh chan<- *relayer.SessionTreeBatch,
) channel.MapFn[*relayer.SessionTreeBatch, either.SessionTreeBatch] {
	return func(
		ctx context.Context,
		sessionTreeBatch *relayer.SessionTreeBatch,
	) (_ either.SessionTreeBatch, skip bool) {
		sessionClaims := []*relayer.SessionClaim{}

		pathBlockHeight := sessionTreeBatch.SessionsEndBlockHeight +
			sessionkeeper.GetSessionGracePeriodBlockCount()
		pathBlock, err := rs.blockQueryClient.Block(ctx, &pathBlockHeight)
		if err != nil {
			return either.Error[*relayer.SessionTreeBatch](err), false
		}

		polylog.Ctx(ctx).Info().
			Int64("session_start_block", pathBlock.Block.Height).
			Msg("submitting claims")

		for _, session := range sessionTreeBatch.SessionTrees {
			// this session should no longer be updated
			claimRoot, err := session.Flush()
			if err != nil {
				return either.Error[*relayer.SessionTreeBatch](err), false
			}

			sessionClaims = append(sessionClaims, &relayer.SessionClaim{
				RootHash:      claimRoot,
				SessionHeader: session.GetSessionHeader(),
			})
		}

		if err := rs.supplierClient.CreateClaims(ctx, sessionClaims); err != nil {
			failedCreateClaimsSessionsPublishCh <- sessionTreeBatch
			return either.Error[*relayer.SessionTreeBatch](err), false
		}

		return either.Success(sessionTreeBatch), false
	}
}
