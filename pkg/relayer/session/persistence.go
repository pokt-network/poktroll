package session

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// populateSessionTreeMap reconstructs the in-memory session tree map from previously persisted sessions.
// This method:
// 1. Retrieves all persisted session metadata from storage
// 2. Validates each session against current blockchain height to determine its lifecycle state
// 3. Deletes expired sessions that are past their proof submission window
// 4. Deletes sessions past their claim window that don't have an associated onchain claim
// 5. Retains and loads valid sessions into memory for continued processing
func (rs *relayerSessionsManager) populateSessionTreeMap(ctx context.Context, height int64) error {
	logger := rs.logger.With("method", "populateSessionTreeMap", "height", height)

	sharedParams, err := rs.sharedQueryClient.GetParams(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get shared params")
		return err
	}

	// Retrieve all persisted session metadata for evaluation of their current state
	_, persistedSessions, err := rs.sessionsMetadataStore.GetAll([]byte{}, false)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get persisted sessions")
		return err
	}

	for _, persistedSession := range persistedSessions {
		// Unmarshal the persisted session metadata for processing
		persistedSMT := &prooftypes.PersistedSMT{}
		if err := persistedSMT.Unmarshal(persistedSession); err != nil {
			logger.Error().Err(err).Msg("failed to unmarshal persisted session metadata, skipping")
			continue
		}

		sessionEndHeight := persistedSMT.SessionHeader.SessionEndBlockHeight
		sessionId := persistedSMT.SessionHeader.SessionId
		supplierOperatorAddress := persistedSMT.SupplierOperatorAddress

		// Create a contextual logger for this specific session
		sessionLogger := logger.With(
			"session_id", sessionId,
			"supplier_operator_address", supplierOperatorAddress,
			"session_end_height", sessionEndHeight,
		)

		// Session lifecycle states and corresponding actions:
		// 1. Active and accepting relays: Retain session for processing new relays
		// 2. Within claim window: Retain session as it's in the process of creating a claim
		// 3. Past claim window with onchain claim: Retain for proof submission
		// 4. Past claim window without onchain claim: Delete as it's too late for settlement
		// 5. Past proof window: Delete as the session is completely expired

		claimWindowCloseHeight := sharedtypes.GetClaimWindowCloseHeight(sharedParams, sessionEndHeight)
		proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(sharedParams, sessionEndHeight)

		// Check if session is expired (past proof submission window)
		// These sessions can no longer reach settlement and should be deleted
		if height > proofWindowCloseHeight {
			if storeErr := rs.deletePersistedSessionTree(persistedSMT); storeErr != nil {
				sessionLogger.Error().Err(storeErr).Msg("failed to delete outdated session tree, skipping")
				continue
			}

			sessionLogger.Info().
				Int64("proof_window_close", proofWindowCloseHeight).
				Int64("current_height", height).
				Msg("Session deleted: expired past proof submission window")

			continue
		}

		// For sessions past the claim window, check if a claim exists onchain
		// If no claim exists, the session can't progress to settlement and should be deleted
		var claim *prooftypes.Claim
		if height > claimWindowCloseHeight {
			claim, queryErr := rs.proofQueryClient.GetClaim(ctx, supplierOperatorAddress, sessionId)
			// No claim was created onchain, delete the session from the store.
			if queryErr != nil || claim == nil {
				if storeErr := rs.deletePersistedSessionTree(persistedSMT); storeErr != nil {
					sessionLogger.Error().Err(storeErr).Msg("failed to delete outdated session without claim, skipping")
					continue
				}

				sessionLogger.Info().Msg("deleting session has no claim created onchain, deleting session metadata")
				continue
			}
		}

		// The session has still a chance to reach settlement.
		// Add it to the sessionTrees map of blockHeight->sessionId->supplierOperatorAddress->sessionTree
		sessionTreesWithEndHeight, ok := rs.sessionsTrees[sessionEndHeight]

		// If there is no map for sessions at the sessionEndHeight, create one.
		if !ok {
			sessionTreesWithEndHeight = make(map[string]map[string]relayer.SessionTree)
			rs.sessionsTrees[sessionEndHeight] = sessionTreesWithEndHeight
		}

		// Get the sessionTreeWithSessionId for the given session.
		sessionTreeWithSessionId, ok := sessionTreesWithEndHeight[sessionId]

		// If there is no map for session trees with the session id, create one.
		if !ok {
			sessionTreeWithSessionId = make(map[string]relayer.SessionTree)
			sessionTreesWithEndHeight[sessionId] = sessionTreeWithSessionId
		}

		_, ok = sessionTreeWithSessionId[supplierOperatorAddress]
		if ok {
			sessionLogger.Error().Msg("the session tree already exists, skipping")
			continue
		}

		sessionTree, treeErr := ImportSessionTree(persistedSMT, claim, rs.storesDirectory, rs.logger)
		if treeErr != nil {
			sessionLogger.Error().Err(treeErr).Msg("failed to import session tree")
			continue
		}

		sessionTreeWithSessionId[supplierOperatorAddress] = sessionTree
	}

	return nil
}

// deletePersistedSessionTree deletes the persisted session tree and its metadata.
func (rs *relayerSessionsManager) deletePersistedSessionTree(persistedSMT *prooftypes.PersistedSMT) error {
	supplierOperatorAddress := persistedSMT.SupplierOperatorAddress
	sessionId := persistedSMT.SessionHeader.SessionId
	sessionEndHeight := persistedSMT.SessionHeader.SessionEndBlockHeight
	logger := rs.logger.With(
		"method", "deletePersistedSessionTree",
		"supplier_operator_address", supplierOperatorAddress,
		"session_id", sessionId,
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
	if err := rs.sessionsMetadataStore.Delete([]byte(sessionId)); err != nil {
		logger.Error().Err(err).Msg("failed to delete outdated session metadata")
		return err
	}

	delete(rs.sessionsTrees[sessionEndHeight][supplierOperatorAddress], sessionId)

	return nil
}

// persistSessionMetadata persists the session metadata to the store.
// It is used to persist the session metadata after a session has been created.
func (rs *relayerSessionsManager) persistSessionMetadata(sessionTree relayer.SessionTree) error {
	sessionId := sessionTree.GetSessionHeader().SessionId
	sessionEndHeight := sessionTree.GetSessionHeader().SessionEndBlockHeight
	supplierOperatorAddress := sessionTree.GetSupplierOperatorAddress()

	persistedSMT := &prooftypes.PersistedSMT{
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

	metadataBz, err := persistedSMT.Marshal()
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal metadata")
		return err
	}

	if err := rs.sessionsMetadataStore.Set([]byte(sessionId), metadataBz); err != nil {
		logger.Error().Err(err).Msg("failed to persist session metadata")
		return err
	}

	return nil
}
