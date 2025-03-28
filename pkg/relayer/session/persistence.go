package session

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gogo/status"
	"github.com/pokt-network/poktroll/pkg/relayer"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"google.golang.org/grpc/codes"
)

// populateSessionTreeMap reconstructs the in-memory session tree map from previously persisted sessions.
// This method:
// 1. Retrieves all persisted session metadata from storage
// 2. Validates each session against current blockchain height to determine its lifecycle state
// 3. Deletes expired sessions whose proof submission window has elapsed
// 4. Deletes unclaimed sessions whose claim creation window has elapsed
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

		// Session lifecycle states and corresponding actions:
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
		if height >= claimWindowOpenHeight {
			foundClaim, err := rs.proofQueryClient.GetClaim(ctx, supplierOperatorAddress, sessionId)
			if err != nil && status.Convert(err).Code() != codes.NotFound {
				sessionLogger.Error().Err(err).Msgf("failed to query claim for session %s", sessionId)
				continue
			} else {
				claim = foundClaim.(*prooftypes.Claim)
			}
		}

		// Check if session is expired (past proof submission window)
		// These sessions can no longer reach settlement and should be deleted
		if height > proofWindowCloseHeight {
			// A claim was created onchain, but the session is outdated.
			if claim != nil {
				sessionLogger.Warn().
					Msg("Session is outdated, but a claim exists onchain, expecting the supplier to be slashed")
			}

			if storeErr := rs.deletePersistedSessionTree(persistedSMT); storeErr != nil {
				sessionLogger.Error().Err(storeErr).Msg("failed to delete outdated session tree, skipping")
				continue
			}

			sessionLogger.Info().Msg("Session deleted: expired past proof submission window")

			continue
		}

		// For sessions past the claim window, check if a claim exists onchain
		// If no claim exists, the session can't progress to settlement and should be deleted
		if height > claimWindowCloseHeight {
			// No claim was created onchain, delete the session from the store.
			if claim == nil {
				if storeErr := rs.deletePersistedSessionTree(persistedSMT); storeErr != nil {
					sessionLogger.Error().Err(storeErr).Msg("failed to delete outdated session without claim, skipping")
					continue
				}

				sessionLogger.Info().Msg("deleting session has no claim created onchain, deleting session metadata")
				continue
			}
		}

		// The session has still a chance to reach settlement.
		sessionTree, treeErr := ImportSessionTree(persistedSMT, claim, rs.storesDirectory, rs.logger)
		if treeErr != nil {
			sessionLogger.Error().Err(treeErr).Msg("failed to import session tree")
			continue
		}

		// Insert the session tree into the in-memory map to be processed by the observable pipeline.
		if ok := rs.insertSessionTree(sessionTree); !ok {
			sessionLogger.Error().Msg("the session tree already exists, skipping")
		}
	}

	return nil
}

// deletePersistedSessionTree deletes the persisted session tree key/value store and its metadata.
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
// It is used to persist the session metadata using the sessionId as key,
// after a session has been created.
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

// insertSessionTree adds the given session to the sessionTrees map of
// blockHeight->sessionId->supplierOperatorAddress->sessionTree
func (rs *relayerSessionsManager) insertSessionTree(sessionTree relayer.SessionTree) bool {
	sessionEndHeight := sessionTree.GetSessionHeader().SessionEndBlockHeight
	sessionId := sessionTree.GetSessionHeader().SessionId
	supplierOperatorAddress := sessionTree.GetSupplierOperatorAddress()

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

	if _, ok = sessionTreeWithSessionId[supplierOperatorAddress]; ok {
		return false
	}

	sessionTreeWithSessionId[supplierOperatorAddress] = sessionTree
	return true
}
