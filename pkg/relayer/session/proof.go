package session

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
	proofkeeper "github.com/pokt-network/poktroll/x/proof/keeper"
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
	failSubmitProofsSessionsCh chan<- []relayer.SessionTree,
) []relayer.SessionTree {
	// Given the sessionTrees are grouped by their sessionEndHeight, we can use the
	// first one from the group to calculate the earliest height for proof submission.
	sessionEndHeight := sessionTrees[0].GetSessionHeader().GetSessionEndBlockHeight()

	// TODO_TECHDEBT(#543): We don't really want to have to query the params for every method call.
	// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
	// to get the most recently (asynchronously) observed (and cached) value.
	// TODO_BLOCKER(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
	// we should be using the value that the params had for the session which includes queryHeight.
	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		failSubmitProofsSessionsCh <- sessionTrees
		return nil
	}

	submitProofsWindowOpenHeight := shared.GetProofWindowOpenHeight(sharedParams, sessionEndHeight)

	// we wait for submitProofsWindowOpenHeight to be received before proceeding since we need its hash
	rs.logger.Info().
		Int64("submitProofsWindowOpenHeight", submitProofsWindowOpenHeight).
		Msg("waiting & blocking for global earliest proof submission height")

	// sessionPathBlock is the block that will have its hash used as the
	// source of entropy for all the session trees in that batch, waiting for it to
	// be received before proceeding.
	sessionPathBlockHeight := shared.GetSessionGracePeriodEndHeight(sharedParams, sessionEndHeight)
	sessionPathBlock := rs.waitForBlock(ctx, sessionPathBlockHeight)
	_ = rs.waitForBlock(ctx, submitProofsWindowOpenHeight)

	// Generate proofs for all sessionTrees concurrently while waiting for the
	// earliest submitProofsHeight (pseudorandom submission distribution) to be reached.
	// Use a channel to block until all proofs for the sessionTrees have been generated.
	proofsGeneratedCh := make(chan []relayer.SessionTree)
	defer close(proofsGeneratedCh)
	go rs.goProveClaims(
		ctx,
		sessionTrees,
		sessionPathBlock,
		proofsGeneratedCh,
		failSubmitProofsSessionsCh,
	)

	// Wait for the earliest submitProofsHeight to be reached before proceeding.
	earliestSubmitProofsHeight := protocol.GetEarliestSubmitProofHeight(ctx, sessionPathBlock)
	_ = rs.waitForBlock(ctx, earliestSubmitProofsHeight)

	// Once the earliest submitProofsHeight has been reached, and all proofs have
	// been generated, return the sessionTrees that have been successfully proven
	// to be submitted on-chain.
	return <-proofsGeneratedCh
}

// newMapProveSessionsFn returns a new MapFn that submits proofs on the given
// session number. Any session which encounters errors while submitting a proof
// is sent on the failedSubmitProofSessions channel.
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

		// Map key is the supplier address.
		sessionProofs := map[string][]*relayer.SessionProof{}
		for _, session := range sessionTrees {
			supplierAddr := session.GetSupplierAddress().String()
			sessionProofs[supplierAddr] = append(sessionProofs[supplierAddr], &relayer.SessionProof{
				ProofBz:         session.GetProofBz(),
				SessionHeader:   session.GetSessionHeader(),
				SupplierAddress: *session.GetSupplierAddress(),
			})
		}

		// Submit proofs for each supplier address in `sessionTrees`.
		for supplierAddr := range sessionProofs {
			// SubmitProof ensures on-chain proof inclusion so we can safely prune the tree.
			supplierClient, ok := rs.supplierClients.SupplierClients[supplierAddr]
			if !ok {
				return either.Error[[]relayer.SessionTree](ErrSupplierClientNotFound), false
			}
			if err := supplierClient.SubmitProofs(ctx, sessionProofs[supplierAddr]); err != nil {
				failedSubmitProofSessionsCh <- sessionTrees
				return either.Error[[]relayer.SessionTree](err), false
			}
		}

		for _, sessionTree := range sessionTrees {
			// Prune the session tree since the proofs have already been submitted.
			if err := sessionTree.Delete(); err != nil {
				return either.Error[[]relayer.SessionTree](err), false
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
		path := proofkeeper.GetPathForProof(
			sessionPathBlock.Hash(),
			sessionTree.GetSessionHeader().GetSessionId(),
		)

		// If the proof cannot be generated, add the sessionTree to the failedProofs.
		if _, err := sessionTree.ProveClosest(path); err != nil {
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
