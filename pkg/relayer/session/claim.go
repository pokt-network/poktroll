package session

import (
	"context"
	"fmt"
	"slices"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/observable/filter"
	"github.com/pokt-network/poktroll/pkg/observable/logging"
	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Cumulative (observed) gas fees for creating a single claim and submitting a single proof:
// - Gas price at time of observance: 0.001uPOKT
// - Value obtained empirically by observing logs during load testing
// - Value may change as network parameters change
// - This value is a function of the claim & proof message sizes
//
// TODO(@bryanchriswhite, #1507): ClaimAndProofGasCost value should be a function of
// the biggest Relay (in num of bytes) and tx_size_cost_per_byte auth module param.
// There should be a two step approach to this:
// 1. Choose a reasonable (emperically observed) p90 of claim & proof sizes across most chains
// 2. TODO_FUTURE: Compute the gas cost dynamically based on the size of the branch being proven.
var ClaimAndProofGasCost = sdktypes.NewInt64Coin(pocket.DenomuPOKT, 10_000)

// createClaims maps over the sessionsToClaimObs observable. For each claim batch, it:
// 1. Calculates the earliest block height at which it is safe to CreateClaims
// 2. Waits for said block and creates the claims onchain
// 3. Maps errors to a new observable and logs them
// 4. Returns an observable of the successfully claimed sessions
// It DOES NOT BLOCK as map operations run in their own goroutines.
func (rs *relayerSessionsManager) createClaims(
	ctx context.Context,
	supplierClient client.SupplierClient,
	sessionsToClaimObs observable.Observable[[]relayer.SessionTree],
) observable.Observable[[]relayer.SessionTree] {
	failedCreateClaimSessionsObs, failedCreateClaimSessionsPublishCh :=
		channel.NewObservable[[]relayer.SessionTree]()

	// Map sessionsToClaimObs to a new observable of the same type which is notified
	// when the session is eligible to be claimed.
	sessionsWithOpenClaimWindowObs := channel.Map(
		ctx, sessionsToClaimObs,
		rs.mapWaitForEarliestCreateClaimsHeight(failedCreateClaimSessionsPublishCh),
	)

	// Map sessionsWithOpenClaimWindowObs to a new observable of an either type,
	// populated with the session or an error, which is notified after the session
	// claims have been created or an error has been encountered, respectively.
	eitherClaimedSessionsObs := channel.Map(
		ctx, sessionsWithOpenClaimWindowObs,
		rs.newMapClaimSessionsFn(supplierClient, failedCreateClaimSessionsPublishCh),
	)

	// TODO_TECHDEBT: pass failed create claim sessions to some retry mechanism.
	// TODO_IMPROVE: It may be useful for the retry mechanism which consumes the
	// observable which corresponds to failSubmitProofsSessionsCh to have a
	// reference to the error which caused the proof submission to fail.
	// In this case, the error may not be persistent.
	logging.LogErrors(ctx, filter.EitherError(ctx, eitherClaimedSessionsObs))

	// Delete failed session trees so they don't get claimed again.
	channel.ForEach(ctx, failedCreateClaimSessionsObs, rs.deleteSessionTrees)

	// Map eitherClaimedSessions to a new observable of []relayer.SessionTree
	// which is notified when the corresponding claims creation succeeded.
	return filter.EitherSuccess(ctx, eitherClaimedSessionsObs)
}

// mapWaitForEarliestCreateClaimsHeight returns a new MapFn that adds a delay
// between being notified and notifying.
// It calculates and waits for the earliest block height, allowed by the protocol,
// at which claims can be created for the given session number, then emits the
// session **at that moment**.
func (rs *relayerSessionsManager) mapWaitForEarliestCreateClaimsHeight(
	failedCreateClaimsSessionsPublishCh chan<- []relayer.SessionTree,
) channel.MapFn[[]relayer.SessionTree, []relayer.SessionTree] {
	return func(
		ctx context.Context,
		sessionTrees []relayer.SessionTree,
	) (_ []relayer.SessionTree, skip bool) {
		sessionTreesToClaim := rs.waitForEarliestCreateClaimsHeight(
			ctx, sessionTrees, failedCreateClaimsSessionsPublishCh,
		)
		if sessionTreesToClaim == nil {
			return nil, true
		}

		return sessionTreesToClaim, false
	}
}

// waitForEarliestCreateClaimsHeight calculates and waits for (blocking until) the
// earliest block height, allowed by the protocol, at which claims can be created
// for a session with the given sessionEndHeight. It is calculated relative to
// sessionEndHeight using onchain governance parameters and randomized input.
// IT IS A BLOCKING FUNCTION.
func (rs *relayerSessionsManager) waitForEarliestCreateClaimsHeight(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
	failedCreateClaimsSessionsCh chan<- []relayer.SessionTree,
) []relayer.SessionTree {
	// Check if sessionTrees is empty to prevent index out of bounds errors
	if len(sessionTrees) == 0 {
		rs.logger.Warn().Msg("received empty sessionTrees array")
		return nil
	}

	// Given the sessionTrees are grouped by their sessionEndHeight, we can use the
	// first one from the group to calculate the earliest height for claim creation.
	sessionEndHeight := sessionTrees[0].GetSessionHeader().GetSessionEndBlockHeight()
	supplierOperatorAddr := sessionTrees[0].GetSupplierOperatorAddress()

	logger := rs.logger.With(
		"session_end_height", sessionEndHeight,
		"supplier_operator_address", supplierOperatorAddr,
	)

	// TODO_MAINNET(#543): We don't really want to have to query the params for every method call.
	// Once `ModuleParamsClient` is implemented, use its replay observable's `#Last()` method
	// to get the most recently (asynchronously) observed (and cached) value.
	// TODO_MAINNET(@bryanchriswhite,#543): We also don't really want to use the current value of the params. Instead,
	// we should be using the value that the params had for the session which includes queryHeight.
	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get shared params")
		failedCreateClaimsSessionsCh <- sessionTrees
		return nil
	}

	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)

	// we wait for claimWindowOpenHeight to be received before proceeding since we need its hash
	// to know where this servicer's claim submission window opens.
	logger = logger.With("claim_window_open_height", claimWindowOpenHeight)
	logger.Info().Msg("waiting & blocking until the earliest claim commit height offset seed block height")

	// The block that'll be used as a source of entropy for which branch(es) to
	// prove should be deterministic and use onchain governance params.
	claimsWindowOpenBlock := rs.waitForBlock(ctx, claimWindowOpenHeight)
	if claimsWindowOpenBlock == nil {
		logger.Warn().Msg("failed to observe earliest claim commit height offset seed block height")
		failedCreateClaimsSessionsCh <- sessionTrees
		return nil
	}

	logger = logger.With("claim_window_open_block_hash", fmt.Sprintf("%x", claimsWindowOpenBlock.Hash()))
	logger.Info().Msg("observed earliest claim commit height offset seed block height")

	// Get the earliest claim commit height for this supplier.
	earliestSupplierClaimsCommitHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		sharedParams,
		sessionEndHeight,
		claimsWindowOpenBlock.Hash(),
		supplierOperatorAddr,
	)

	logger = logger.With("earliest_claim_commit_height", earliestSupplierClaimsCommitHeight)
	logger.Info().Msg("waiting & blocking until the earliest claim commit height for this supplier")

	// Wait for the earliestSupplierClaimsCommitHeight to be reached before proceeding.
	// This waiting is implemented using a goroutine and a buffered channel to enable
	// concurrent processing that ensures:
	// 1. The main process blocks until the required block height is observed.
	// 2. During this waiting period, the system efficiently creates claims in parallel.
	// 3. Execution continues only after both conditions are satisfied: the target block
	//    height is reached AND the claims creation is complete.

	// Concurrently wait for the target block height to be observed.
	// This operation is non-blocking to allow for claims creation to proceed in parallel.
	blockObserved := make(chan struct{}, 1)
	go func() {
		_ = rs.waitForBlock(ctx, earliestSupplierClaimsCommitHeight)
		logger.Info().Msgf("observed earliest claim commit height %d", earliestSupplierClaimsCommitHeight)

		close(blockObserved)
	}()

	// Create claims for the given sessionTrees while waiting for the target block height.
	// TODO_POST_MAINNET(red-0ne): Support skipping sessionTrees that have already their
	// claim created and are ready to be proven.
	claimsFlushed, failedClaims := rs.createClaimRoots(sessionTrees)
	if len(failedClaims) > 0 {
		logger.Warn().Msgf("failed to create claims for %d session trees", len(failedClaims))
		failedCreateClaimsSessionsCh <- failedClaims
	}

	// Block until the target block height is observed.
	<-blockObserved
	return claimsFlushed
}

// newMapClaimSessionsFn returns a new MapFn that creates claims for the given
// session number. Any session which encounters an error while creating a claim
// is sent on the failedCreateClaimSessions channel.
func (rs *relayerSessionsManager) newMapClaimSessionsFn(
	supplierClient client.SupplierClient,
	failedCreateClaimsSessionsPublishCh chan<- []relayer.SessionTree,
) channel.MapFn[[]relayer.SessionTree, either.SessionTrees] {
	return func(
		ctx context.Context,
		sessionTrees []relayer.SessionTree,
	) (_ either.SessionTrees, skip bool) {
		// TODO_POST_MAINNET(red-0ne): Support skipping sessionTrees that have already their
		// claim created and are ready to be proven.
		if len(sessionTrees) == 0 {
			return either.Success(sessionTrees), false
		}
		sessionEndHeight := sessionTrees[0].GetSessionHeader().GetSessionEndBlockHeight()
		supplierOperatorAddress := sessionTrees[0].GetSupplierOperatorAddress()

		logger := rs.logger.With(
			"session_end_height", sessionEndHeight,
			"supplier_operator_address", supplierOperatorAddress,
		)

		// Filter out the session trees that the supplier operator can afford to claim.
		claimableSessionTrees, err := rs.payableProofsSessionTrees(ctx, sessionTrees)
		if err != nil {
			failedCreateClaimsSessionsPublishCh <- sessionTrees
			logger.Error().Err(err).Msg("failed to calculate payable proofs session trees")
			return either.Error[[]relayer.SessionTree](err), false
		}

		// If the supplier operator cannot afford to claim any of the session trees, then:
		// 1. Skip claim creation
		// 2. Return an empty slice of claimable session trees.
		// DEV_NOTE: This is a common case when the supplier operator has insufficient funds.
		if len(claimableSessionTrees) == 0 {
			err = fmt.Errorf(
				"supplier operator %q cannot afford to claim any of the (%d) session trees. ❗ MAKE SURE TO TOP UP YOUR SUPPLIER'S BALANCE ❗",
				sessionTrees[0].GetSupplierOperatorAddress(),
				len(sessionTrees),
			)
			logger.Warn().Msgf("no claimable session trees, skipping claims creation: %v", err)

			// Avoid submitting transactions with no claim messages.
			return either.Error[[]relayer.SessionTree](err), false
		}

		claimMsgs := make([]client.MsgCreateClaim, len(claimableSessionTrees))
		for idx, sessionTree := range claimableSessionTrees {
			claimMsgs[idx] = &prooftypes.MsgCreateClaim{
				RootHash:                sessionTree.GetClaimRoot(),
				SessionHeader:           sessionTree.GetSessionHeader(),
				SupplierOperatorAddress: sessionTree.GetSupplierOperatorAddress(),
			}
		}

		// All session trees in the batch share the same sessionEndHeight, so we
		// can use the first one to calculate the proof window close height.
		//
		// TODO_REFACTOR(@red-0ne): Pass a richer type to the function instead of []SessionTrees to:
		// - Avoid making assumptions about shared properties
		// - Eliminate constant queries for sharedParams
		sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
		if err != nil {
			failedCreateClaimsSessionsPublishCh <- sessionTrees
			logger.Error().Err(err).Msg("failed to get shared params")
			return either.Error[[]relayer.SessionTree](err), false
		}
		claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(sharedParams, sessionEndHeight)

		// Create claims for each supplier operator address in `sessionTrees`.
		if err := supplierClient.CreateClaims(ctx, claimWindowCloseHeight, claimMsgs...); err != nil {
			failedCreateClaimsSessionsPublishCh <- claimableSessionTrees
			logger.Error().Err(err).Msg("failed to create claims")
			return either.Error[[]relayer.SessionTree](err), false
		}

		return either.Success(claimableSessionTrees), false
	}
}

// CreateClaimRoots creates the claim roots corresponding to the given sessionTrees.
func (rs *relayerSessionsManager) createClaimRoots(
	sessionTrees []relayer.SessionTree,
) (flushedClaims []relayer.SessionTree, failedClaims []relayer.SessionTree) {
	for _, sessionTree := range sessionTrees {
		// This session should no longer be updated
		if _, err := sessionTree.Flush(); err != nil {
			rs.logger.Error().Err(err).Msg("failed to flush session")
			failedClaims = append(failedClaims, sessionTree)
			continue
		}

		flushedClaims = append(flushedClaims, sessionTree)
	}

	return flushedClaims, failedClaims
}

// payableProofsSessionTrees returns the session trees that the supplier operator
// can afford to claim (i.e. pay the fee for submitting a proof).
// The session trees are sorted from the most rewarding to the least rewarding to
// ensure optimal rewards in the case of insufficient funds.
// Note that all sessionTrees are associated with the same supplier operator address.
func (rs *relayerSessionsManager) payableProofsSessionTrees(
	ctx context.Context,
	sessionTrees []relayer.SessionTree,
) ([]relayer.SessionTree, error) {
	// Check if sessionTrees is empty to prevent index out of bounds errors
	if len(sessionTrees) == 0 {
		return sessionTrees, nil
	}

	supplierOperatorAddress := sessionTrees[0].GetSupplierOperatorAddress()
	logger := rs.logger.With(
		"supplier_operator_address", supplierOperatorAddress,
	)

	proofParams, err := rs.proofQueryClient.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	// Account for the gas cost of creating a claim and submitting a proof.
	// This accounts for onchain fees (pocket specific) and gas costs (network wide).
	proofSubmissionFee := proofParams.GetProofSubmissionFee()
	claimAndProofSubmissionCost := proofSubmissionFee.Add(ClaimAndProofGasCost)

	supplierOperatorBalanceCoin, err := rs.bankQueryClient.GetBalance(
		ctx,
		supplierOperatorAddress,
	)
	if err != nil {
		return nil, err
	}

	// Sort the session trees by the sum of the claim root to ensure that the
	// most rewarding claims are claimed first.
	slices.SortFunc(sessionTrees, func(a, b relayer.SessionTree) int {
		rootA := a.GetClaimRoot()
		sumA, errA := smt.MerkleSumRoot(rootA).Sum()
		if errA != nil {
			logger.With(
				"session_id", a.GetSessionHeader().GetSessionId(),
				"claim_root", fmt.Sprintf("%x", rootA),
			).Error().Err(errA).Msg("failed to calculate sum of claim root, assuming 0")
			sumA = 0
		}

		rootB := b.GetClaimRoot()
		sumB, errB := smt.MerkleSumRoot(rootB).Sum()
		if errB != nil {
			logger.With(
				"session_id", a.GetSessionHeader().GetSessionId(),
				"claim_root", fmt.Sprintf("%x", rootA),
			).Error().Err(errB).Msg("failed to calculate sum of claim root, assuming 0")
			sumB = 0
		}

		// Sort in descending order.
		return int(sumB - sumA)
	})

	claimableSessionTrees := []relayer.SessionTree{}
	for _, sessionTree := range sessionTrees {
		// Supplier CAN afford to claim the session.
		// Add it to the claimableSessionTrees slice.
		supplierCanAffordClaimAndProofFees := supplierOperatorBalanceCoin.IsGTE(claimAndProofSubmissionCost)

		claimLogger := logger.With(
			"session_id", sessionTree.GetSessionHeader().GetSessionId(),
		)

		claimReward, err := rs.getClaimRewardCoin(ctx, sessionTree)
		if err != nil {
			claimLogger.Error().Err(err).Msg("failed to calculate claim reward")
			return nil, err
		}

		isClaimProfitable := claimReward.IsGT(ClaimAndProofGasCost)

		if supplierCanAffordClaimAndProofFees && isClaimProfitable {
			claimableSessionTrees = append(claimableSessionTrees, sessionTree)
			newSupplierOperatorBalanceCoin := supplierOperatorBalanceCoin.Sub(claimAndProofSubmissionCost)
			supplierOperatorBalanceCoin = &newSupplierOperatorBalanceCoin

			estimatedClaimProfit := claimReward.Sub(ClaimAndProofGasCost)
			claimLogger.Info().Msgf(
				"💲 Processing profitable claim — estimated submission cost 💸: %s, reward 🎁: %s, estimated profit 💰: %s",
				claimAndProofSubmissionCost, claimReward, estimatedClaimProfit,
			)

			continue
		}

		// Supplier CANNOT afford to claim the session.
		// Delete the session tree from the relayer sessions and the KVStore since
		// it won't be claimed due to insufficient funds.
		rs.removeFromRelayerSessions(sessionTree)
		if err := sessionTree.Delete(); err != nil {
			claimLogger.Error().Err(err).Msg("failed to delete session tree")
		}

		if !isClaimProfitable {
			// Calculate how unprofitable the claim is
			unprofitableAmount := ClaimAndProofGasCost.Sub(claimReward)
			// Log a warning with details about how unprofitable the claim is in plain English
			claimLogger.Warn().Msgf(
				"⚠️ Aborting claim — cost exceeds reward by %s (reward: %s). 🧹 Cleaning up session state.",
				unprofitableAmount, claimReward,
			)
		}

		if !supplierCanAffordClaimAndProofFees {
			// Log a warning of any session that the supplier operator cannot afford to claim.
			claimLogger.Warn().Msgf(
				"⚠️ Aborting claim — supplier operator has insufficient funds to submit claim & proof (cost: %s, balance: %s). 🧹 Cleaning up session tree.",
				claimAndProofSubmissionCost, supplierOperatorBalanceCoin,
			)
		}
	}

	if len(claimableSessionTrees) < len(sessionTrees) {
		logger.Warn().Msgf(
			"⚠️ Supplier operator %q can only process %d out of %d claims. 💰 Prioritizing most profitable ones.",
			supplierOperatorAddress, len(claimableSessionTrees), len(sessionTrees),
		)
	}

	return claimableSessionTrees, nil
}

// getClaimRewardCoin calculates the number of uPOKT the supplier claimed for the particular session.
// It uses the serviceID from the tree's session header and queries onchain data for downstream calculations
func (rs *relayerSessionsManager) getClaimRewardCoin(
	ctx context.Context,
	sessionTree relayer.SessionTree,
) (sdktypes.Coin, error) {
	sessionHeader := sessionTree.GetSessionHeader()
	serviceId := sessionHeader.GetServiceId()

	// Create a claim object to calculate the claim reward.
	claim := claimFromSessionTree(sessionTree)

	relayMiningDifficulty, err := rs.serviceQueryClient.GetServiceRelayDifficulty(ctx, serviceId)
	if err != nil {
		return sdktypes.Coin{}, err
	}

	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		return sdktypes.Coin{}, err
	}

	return claim.GetClaimeduPOKT(*sharedParams, relayMiningDifficulty)
}
