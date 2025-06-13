package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// loadSessionTreeMap reconstructs the in-memory session tree map from previously persisted sessions.
//
// It was implemented in #1140 to ensure that RelayMiner can recover from restarts
// and crashes and account for the following:
// - Submit a proof if a prior claim was created before the crash
// - Continue earning rewards for an existing claim that already mined relays
//
// This method:
// 1. Retrieves all persisted session metadata from storage (i.e. from disk)
// 2. Validates each session against current blockchain height to determine its lifecycle state (i.e. active, expired, etc.)
// 3. Deletes expired sessions whose proof submission window has elapsed
// 4. Deletes unclaimed sessions whose claim creation window has elapsed
func (rs *relayerSessionsManager) loadSessionTreeMap(ctx context.Context, height int64) error {
	logger := rs.logger.With("method", "populateSessionTreeMap", "height", height)

	// Retrieve the shared onchain parameters
	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get shared params")
		return err
	}

	// Retrieve all persisted session metadata for evaluation of their current state
	_, persistedSessions, err := rs.sessionSMTStore.GetAll([]byte{}, false)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get persisted sessions")
		return err
	}

	if len(persistedSessions) > 0 {
		logger.Info().Msgf("about to load %d persisted sessions into memory", len(persistedSessions))
	}

	for _, persistedSession := range persistedSessions {
		// Unmarshal the persisted session metadata for processing
		sessionSMT := &prooftypes.SessionSMT{}
		if err := sessionSMT.Unmarshal(persistedSession); err != nil {
			logger.Error().Err(err).Msg("failed to unmarshal persisted session metadata, skipping")
			continue
		}

		// Extract the session metadata from the persisted session
		sessionEndHeight := sessionSMT.SessionHeader.SessionEndBlockHeight
		sessionId := sessionSMT.SessionHeader.SessionId
		supplierOperatorAddress := sessionSMT.SupplierOperatorAddress

		// There are 5 session lifecycle states/scenarios.
		// The following outlines what action should be done in each one.
		// 1. Active and accepting relays: Retain session for processing new relays
		// 2. Within claim window: Retain session as it's in the process of creating a claim
		// 3. Past claim window with onchain claim: Retain for proof submission
		// 4. Past claim window without onchain claim: Delete as it's too late to claim
		// 5. Past proof window: Delete as the session is completely expired

		claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)
		claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(sharedParams, sessionEndHeight)
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(sharedParams, sessionEndHeight)

		// Create a contextual logger for this specific session
		sessionLogger := logger.With(
			"session_id", sessionId,
			"supplier_operator_address", supplierOperatorAddress,
			"session_end_height", sessionEndHeight,
			"claim_window_close", claimWindowCloseHeight,
			"proof_window_close", proofWindowCloseHeight,
			"current_height", height,
		)

		var claim *prooftypes.Claim

		// Scenario 2: The claim window has opened.
		// A claim may already exist or may still be created.
		// Try to retrieve the claim (from the chain or cache) associated with the session id and supplier.
		if height >= claimWindowOpenHeight {
			foundClaim, err := rs.proofQueryClient.GetClaim(ctx, supplierOperatorAddress, sessionId)
			// If the query returns an error other than NotFound, log the error and continue.
			if err != nil && status.Convert(err).Code() != codes.NotFound {
				sessionLogger.Error().Err(err).Msgf("failed to query claim for session %s", sessionId)
				continue
			}

			// If the claim was not found, it may still be created.
			if err != nil && status.Convert(err).Code() == codes.NotFound {
				// No claim was found for this session, but it may still be created.
				sessionLogger.Debug().Msgf("claim not found onchain for session %s", sessionId)
				claim = nil
			}

			// A claim was successfully retrieved for this session, use it to determine the session lifecycle state,
			if err == nil && foundClaim != nil {
				var ok bool
				claim, ok = foundClaim.(*prooftypes.Claim)
				if !ok {
					sessionLogger.Error().Msg("failed to cast claim to prooftypes.Claim")
					continue
				}
			}
		}

		// Scenario 5: The proof window has closed.
		// If a claim were to be created and a proof were to be submitted, both must already exist onchain.
		// Check if the session is expired (proof submission window has closed) and prune those sessions from memory an the store.
		if height > proofWindowCloseHeight {
			// A claim was created onchain (i.e. available), but the session is outdated.
			if claim != nil {
				sessionLogger.Warn().
					Msg("Session is outdated, but a claim exists onchain, expecting the supplier to be slashed")
			}

			// Clean up by delete the session tree from the store.
			if storeErr := rs.deletePersistedSessionTree(sessionSMT); storeErr != nil {
				sessionLogger.Error().Err(storeErr).Msg("failed to delete outdated session tree, skipping")
				continue
			}

			sessionLogger.Info().Msg("Session deleted: expired past proof submission window")

			continue
		}

		// Scenarios 3 & 4: The claim window has closed.
		// Check if a claim was created to ensure a proof is submitted (if required) so the RelayMiner earns rewards!
		// If no claim exists onchain, the session can't progress to settlement and should be deleted.
		if height > claimWindowCloseHeight {
			// Scenario 4: The claim window has closed and no claim exists onchain.
			// No claim was created onchain, delete the session from the store.
			if claim == nil {
				if storeErr := rs.deletePersistedSessionTree(sessionSMT); storeErr != nil {
					sessionLogger.Error().Err(storeErr).Msg("failed to delete session WITHOUT onchain claim, skipping")
					continue
				}

				sessionLogger.Info().Msg("deleting session WITHOUT onchain claim")
				continue
			}
			// Scenario 3: The claim window has closed and a claim exists onchain -> may or may not need to submit a proof.
		}

		// Scenarios 2: The claim window is still open.
		// The session has still a chance to reach settlement by creating the claim and submitting the proof.
		sessionTree, treeErr := importSessionTree(sessionSMT, claim, rs.storesDirectory, rs.logger)
		if treeErr != nil {
			sessionLogger.Error().Err(treeErr).Msg("failed to import session tree")
			continue
		}

		// Insert the session tree into the in-memory map to be processed by the observable pipeline.
		// This ensures that Claim Creation & Proof Submissions are processed in the correct order.
		if ok := rs.insertSessionTree(sessionTree); !ok {
			sessionLogger.Error().Msg("the session tree already exists, skipping")
		}
	}

	// Process the sessions that are ready for proof submission here instead of
	// using the normal pipeline which currently does not support skipping the
	// claim creation step.
	rs.proveClaimedSessions(ctx)

	return nil
}

// deletePersistedSessionTree deletes:
// - The full persisted session tree on-disk key/value store
// - The session tree metadata entry from the in-memory store
func (rs *relayerSessionsManager) deletePersistedSessionTree(sessionSMT *prooftypes.SessionSMT) error {
	supplierOperatorAddress := sessionSMT.SupplierOperatorAddress
	sessionId := sessionSMT.SessionHeader.SessionId
	sessionEndHeight := sessionSMT.SessionHeader.SessionEndBlockHeight

	logger := rs.logger.With(
		"method", "deletePersistedSessionTree",
		"supplier_operator_address", supplierOperatorAddress,
		"session_id", sessionId,
		"session_end_height", sessionEndHeight,
	)

	// Delete the corresponding kv store for the session tree.
	storePath := filepath.Join(rs.storesDirectory, supplierOperatorAddress, sessionId)
	if err := os.RemoveAll(storePath); err != nil {
		if os.IsNotExist(err) {
			logger.Warn().Msg("session tree store does not exist, nothing to delete")
			return nil
		}

		logger.Error().Err(err).Msg("failed to delete outdated session tree store")
		return err
	}

	// Delete the persisted session tree metadata
	sessionStoreKey := getSessionStoreKey(supplierOperatorAddress, sessionId)
	if err := rs.sessionSMTStore.Delete(sessionStoreKey); err != nil {
		logger.Error().Err(err).Msg("failed to delete outdated session metadata")
		return err
	}

	delete(rs.sessionsTrees[supplierOperatorAddress][sessionEndHeight], sessionId)

	return nil
}

// persistSessionMetadata persists the session metadata to the store.
// It is used to persist the session metadata using the sessionId as key,
// after a session has been created.
func (rs *relayerSessionsManager) persistSessionMetadata(sessionTree relayer.SessionTree) error {
	supplierOperatorAddress := sessionTree.GetSupplierOperatorAddress()
	sessionId := sessionTree.GetSessionHeader().SessionId
	sessionEndHeight := sessionTree.GetSessionHeader().SessionEndBlockHeight

	sessionSMT := &prooftypes.SessionSMT{
		SessionHeader:           sessionTree.GetSessionHeader(),
		SupplierOperatorAddress: supplierOperatorAddress,
		SmtRoot:                 sessionTree.GetSMSTRoot(),
	}
	logger := rs.logger.With(
		"method", "persistSessionMetadata",
		"supplier_operator_address", supplierOperatorAddress,
		"session_id", sessionId,
		"session_end_height", sessionEndHeight,
	)

	sessionSMTBz, err := sessionSMT.Marshal()
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal metadata")
		return err
	}

	sessionStoreKey := getSessionStoreKey(supplierOperatorAddress, sessionId)
	if err := rs.sessionSMTStore.Set(sessionStoreKey, sessionSMTBz); err != nil {
		logger.Error().Err(err).Msg("failed to persist session metadata")
		return err
	}

	return nil
}

// insertSessionTree adds the given session to the sessionTrees map of
// supplierOperatorAddress->blockHeight->sessionId->sessionTree.
// Return true if the session was inserted, false if it already exists.
func (rs *relayerSessionsManager) insertSessionTree(sessionTree relayer.SessionTree) bool {
	supplierOperatorAddress := sessionTree.GetSupplierOperatorAddress()
	sessionId := sessionTree.GetSessionHeader().SessionId
	sessionEndHeight := sessionTree.GetSessionHeader().SessionEndBlockHeight

	// Get all the suppliersSessionTrees for the given supplier operator
	supplierSessionTrees, ok := rs.sessionsTrees[supplierOperatorAddress]

	// If there is no map for session trees with the supplier operator address, create one.
	if !ok {
		supplierSessionTrees = make(map[int64]map[string]relayer.SessionTree)
		rs.sessionsTrees[supplierOperatorAddress] = supplierSessionTrees
	}

	// Get the session trees for the given sessionEndHeight.
	sessionTreesWithEndHeight, ok := supplierSessionTrees[sessionEndHeight]

	// If there is no map for sessions at the sessionEndHeight, create one.
	if !ok {
		sessionTreesWithEndHeight = make(map[string]relayer.SessionTree)
		supplierSessionTrees[sessionEndHeight] = sessionTreesWithEndHeight
	}

	// Get the sessionTree for the given session id.
	// An already existing session tree means that the the sessionTreeStore has
	// duplicate session trees for the same supplier operator address and session id,
	// which should not happen.
	if _, ok = sessionTreesWithEndHeight[sessionId]; ok {
		rs.logger.Warn().
			Str("session_id", sessionId).
			Str("supplier_operator_address", supplierOperatorAddress).
			Msg("session tree already exists, skipping")
		return false
	}

	sessionTreesWithEndHeight[sessionId] = sessionTree

	return true
}

// proveClaimedSessions processes the claimed sessions and sends them to the
// relayer for proof submission without going through the whole pipeline.
func (rs *relayerSessionsManager) proveClaimedSessions(ctx context.Context) {
	// Iterate over all supplier clients so we can submit proofs for all claimed sessions.
	for supplierOperatorAddress, supplierClient := range rs.supplierClients.SupplierClients {
		// TODO_TECHDEBT(@bryanchriswhite): Close the channel when all the sessions have been processed.
		sessionsToProveObs, sessionsToProvePublishCh := channel.NewObservable[[]relayer.SessionTree]()

		// Start an observable that'll be listening on incoming proofs that can be submitted.
		// Submit all the proofs for the current supplier client
		rs.submitProofs(ctx, supplierClient, sessionsToProveObs)

		// Find all the sessions that have a proof ready to be submitted and publish them to the channel
		// that the observable above is listening on.
		for _, sessionTreesWithEndHeight := range rs.sessionsTrees[supplierOperatorAddress] {
			sessionsToProve := make([]relayer.SessionTree, 0)

			// Loop through all session trees whose session end height is now.
			for _, sessionTree := range sessionTreesWithEndHeight {
				// If the session has its claim root generated, it means that the session
				// has been claimed and is ready to go through the proof submission pipeline.
				if sessionTree.GetClaimRoot() != nil {
					sessionsToProve = append(sessionsToProve, sessionTree)
				}
			}

			// Publish the sessions to the channel so that the observable can process them.
			if len(sessionsToProve) > 0 {
				sessionsToProvePublishCh <- sessionsToProve
			}
		}
	}
}

// getSessionStoreKey constructs the store key for a sessionTree in the form of: supplierOperatorAddress/sessionId.
func getSessionStoreKey(supplierOperatorAddress string, sessionId string) []byte {
	sessionStoreKeyStr := fmt.Sprintf("%s/%s", supplierOperatorAddress, sessionId)
	return []byte(sessionStoreKeyStr)
}
