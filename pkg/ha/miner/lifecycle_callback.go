package miner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// LifecycleCallbackConfig contains configuration for the lifecycle callback.
type LifecycleCallbackConfig struct {
	// SupplierAddress is the supplier this callback is for.
	SupplierAddress string

	// ClaimRetryAttempts is the number of times to retry failed claims.
	ClaimRetryAttempts int

	// ClaimRetryDelay is the delay between retry attempts.
	ClaimRetryDelay time.Duration

	// ProofRetryAttempts is the number of times to retry failed proofs.
	ProofRetryAttempts int

	// ProofRetryDelay is the delay between retry attempts.
	ProofRetryDelay time.Duration
}

// DefaultLifecycleCallbackConfig returns sensible defaults.
func DefaultLifecycleCallbackConfig() LifecycleCallbackConfig {
	return LifecycleCallbackConfig{
		ClaimRetryAttempts: 3,
		ClaimRetryDelay:    2 * time.Second,
		ProofRetryAttempts: 3,
		ProofRetryDelay:    2 * time.Second,
	}
}

// SMSTManager provides SMST operations for claim/proof generation.
// This interface combines what the lifecycle callback needs from both
// SMSTFlusher and SMSTProver.
type SMSTManager interface {
	// FlushTree flushes the SMST for a session and returns the root hash.
	FlushTree(ctx context.Context, sessionID string) (rootHash []byte, err error)

	// GetTreeRoot returns the root hash for an already-flushed session.
	GetTreeRoot(ctx context.Context, sessionID string) (rootHash []byte, err error)

	// ProveClosest generates a proof for the closest leaf to the given path.
	ProveClosest(ctx context.Context, sessionID string, path []byte) (proofBytes []byte, err error)

	// DeleteTree removes the SMST for a session (cleanup after settlement).
	DeleteTree(ctx context.Context, sessionID string) error
}

// SessionQueryClient queries session information from the blockchain.
type SessionQueryClient interface {
	GetSession(ctx context.Context, appAddr, serviceID string, blockHeight int64) (*sessiontypes.Session, error)
}

// LifecycleCallback implements SessionLifecycleCallback to handle claim and proof submission.
// It coordinates SMST operations with transaction submission and uses proper timing spread.
type LifecycleCallback struct {
	logger         polylog.Logger
	config         LifecycleCallbackConfig
	supplierClient client.SupplierClient
	sharedClient   client.SharedQueryClient
	blockClient    client.BlockClient
	sessionClient  SessionQueryClient
	smstManager    SMSTManager
	snapshotMgr    *SMSTSnapshotManager

	// Per-session locks to prevent concurrent claim/proof operations
	sessionLocks   map[string]*sync.Mutex
	sessionLocksMu sync.Mutex
}

// NewLifecycleCallback creates a new lifecycle callback.
func NewLifecycleCallback(
	logger polylog.Logger,
	supplierClient client.SupplierClient,
	sharedClient client.SharedQueryClient,
	blockClient client.BlockClient,
	sessionClient SessionQueryClient,
	smstManager SMSTManager,
	snapshotMgr *SMSTSnapshotManager,
	config LifecycleCallbackConfig,
) *LifecycleCallback {
	if config.ClaimRetryAttempts <= 0 {
		config.ClaimRetryAttempts = 3
	}
	if config.ClaimRetryDelay <= 0 {
		config.ClaimRetryDelay = 2 * time.Second
	}
	if config.ProofRetryAttempts <= 0 {
		config.ProofRetryAttempts = 3
	}
	if config.ProofRetryDelay <= 0 {
		config.ProofRetryDelay = 2 * time.Second
	}

	return &LifecycleCallback{
		logger:         logging.ForSupplierComponent(logger, logging.ComponentLifecycleCallback, config.SupplierAddress),
		config:         config,
		supplierClient: supplierClient,
		sharedClient:   sharedClient,
		blockClient:    blockClient,
		sessionClient:  sessionClient,
		smstManager:    smstManager,
		snapshotMgr:    snapshotMgr,
		sessionLocks:   make(map[string]*sync.Mutex),
	}
}

// getSessionLock returns a per-session lock.
func (lc *LifecycleCallback) getSessionLock(sessionID string) *sync.Mutex {
	lc.sessionLocksMu.Lock()
	defer lc.sessionLocksMu.Unlock()

	lock, exists := lc.sessionLocks[sessionID]
	if !exists {
		lock = &sync.Mutex{}
		lc.sessionLocks[sessionID] = lock
	}
	return lock
}

// removeSessionLock removes a per-session lock.
func (lc *LifecycleCallback) removeSessionLock(sessionID string) {
	lc.sessionLocksMu.Lock()
	defer lc.sessionLocksMu.Unlock()
	delete(lc.sessionLocks, sessionID)
}

// OnSessionActive is called when a new session starts.
// For HA miner, sessions are created on-demand when relays arrive, so this is mostly informational.
func (lc *LifecycleCallback) OnSessionActive(ctx context.Context, snapshot *SessionSnapshot) error {
	lc.logger.Info().
		Str(logging.FieldSessionID, snapshot.SessionID).
		Int64(logging.FieldSessionEndHeight, snapshot.SessionEndHeight).
		Str(logging.FieldServiceID, snapshot.ServiceID).
		Msg("session active")

	return nil
}

// OnSessionNeedsClaim is called when a session needs a claim submitted.
// It waits for the proper timing spread, flushes the SMST, and submits the claim.
func (lc *LifecycleCallback) OnSessionNeedsClaim(ctx context.Context, snapshot *SessionSnapshot) (rootHash []byte, err error) {
	lock := lc.getSessionLock(snapshot.SessionID)
	lock.Lock()
	defer lock.Unlock()

	logger := lc.logger.With(
		logging.FieldSessionID, snapshot.SessionID,
		logging.FieldSessionEndHeight, snapshot.SessionEndHeight,
	)

	logger.Info().
		Int64(logging.FieldCount, snapshot.RelayCount).
		Msg("session needs claim - starting claim process")

	// Get shared params
	sharedParams, err := lc.sharedClient.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared params: %w", err)
	}

	// Wait for claim window to open and get the block hash for timing spread
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, snapshot.SessionEndHeight)
	logger.Info().
		Int64("claim_window_open_height", claimWindowOpenHeight).
		Msg("waiting for claim window to open")

	claimWindowOpenBlock, err := lc.waitForBlock(ctx, claimWindowOpenHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for claim window open: %w", err)
	}

	// Calculate the earliest claim commit height for this supplier (timing spread)
	earliestClaimHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		sharedParams,
		snapshot.SessionEndHeight,
		claimWindowOpenBlock.Hash(),
		snapshot.SupplierOperatorAddress,
	)

	// Record the scheduled claim height for operators
	SetClaimScheduledHeight(snapshot.SupplierOperatorAddress, snapshot.SessionID, float64(earliestClaimHeight))

	logger.Info().
		Int64("earliest_claim_height", earliestClaimHeight).
		Msg("waiting for assigned claim timing")

	// Wait for the earliest claim height (timing spread ensures suppliers don't all submit at once)
	if _, err := lc.waitForBlock(ctx, earliestClaimHeight); err != nil {
		return nil, fmt.Errorf("failed to wait for claim timing: %w", err)
	}

	logger.Info().Msg("claim window timing reached - flushing SMST and submitting claim")

	// Flush the SMST to get the root hash
	rootHash, err = lc.smstManager.FlushTree(ctx, snapshot.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to flush SMST: %w", err)
	}

	// Build the session header
	sessionHeader, err := lc.buildSessionHeader(ctx, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to build session header: %w", err)
	}

	// Calculate timeout height (claim window close)
	claimWindowClose := sharedtypes.GetClaimWindowCloseHeight(sharedParams, snapshot.SessionEndHeight)

	// Build and submit the claim with retries
	claimMsg := &prooftypes.MsgCreateClaim{
		SupplierOperatorAddress: snapshot.SupplierOperatorAddress,
		SessionHeader:           sessionHeader,
		RootHash:                rootHash,
	}

	var lastErr error
	for attempt := 1; attempt <= lc.config.ClaimRetryAttempts; attempt++ {
		if err := lc.supplierClient.CreateClaims(ctx, claimWindowClose, claimMsg); err != nil {
			lastErr = err
			logger.Warn().
				Err(err).
				Int(logging.FieldAttempt, attempt).
				Int(logging.FieldMaxRetry, lc.config.ClaimRetryAttempts).
				Msg("claim submission failed, retrying")

			if attempt < lc.config.ClaimRetryAttempts {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(lc.config.ClaimRetryDelay):
					continue
				}
			}
		} else {
			// Success
			currentBlock := lc.blockClient.LastBlock(ctx)
			blocksAfterWindowOpen := float64(currentBlock.Height() - claimWindowOpenHeight)

			RecordClaimSubmitted(snapshot.SupplierOperatorAddress)
			RecordClaimSubmissionLatency(snapshot.SupplierOperatorAddress, blocksAfterWindowOpen)
			RecordComputeUnitsClaimed(snapshot.SupplierOperatorAddress, snapshot.ServiceID, float64(snapshot.TotalComputeUnits))

			logger.Info().
				Int("root_hash_len", len(rootHash)).
				Int64("blocks_after_window", int64(blocksAfterWindowOpen)).
				Msg("claim submitted successfully")

			// Update snapshot manager
			if lc.snapshotMgr != nil {
				if updateErr := lc.snapshotMgr.OnSessionClaimed(ctx, snapshot.SessionID, rootHash); updateErr != nil {
					logger.Warn().Err(updateErr).Msg("failed to update snapshot after claim")
				}
			}

			return rootHash, nil
		}
	}

	RecordClaimError(snapshot.SupplierOperatorAddress, "exhausted_retries")
	return nil, fmt.Errorf("claim submission failed after %d attempts: %w", lc.config.ClaimRetryAttempts, lastErr)
}

// OnSessionNeedsProof is called when a session needs a proof submitted.
// It waits for the proper timing spread, generates the proof, and submits it.
func (lc *LifecycleCallback) OnSessionNeedsProof(ctx context.Context, snapshot *SessionSnapshot) error {
	lock := lc.getSessionLock(snapshot.SessionID)
	lock.Lock()
	defer lock.Unlock()

	logger := lc.logger.With(
		logging.FieldSessionID, snapshot.SessionID,
		logging.FieldSessionEndHeight, snapshot.SessionEndHeight,
	)

	logger.Info().Msg("session needs proof - starting proof process")

	// Get shared params
	sharedParams, err := lc.sharedClient.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get shared params: %w", err)
	}

	// Wait for proof window to open
	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(sharedParams, snapshot.SessionEndHeight)
	logger.Info().
		Int64("proof_window_open_height", proofWindowOpenHeight).
		Msg("waiting for proof window to open")

	proofWindowOpenBlock, err := lc.waitForBlock(ctx, proofWindowOpenHeight)
	if err != nil {
		return fmt.Errorf("failed to wait for proof window open: %w", err)
	}

	// Calculate the earliest proof commit height for this supplier (timing spread)
	earliestProofHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		snapshot.SessionEndHeight,
		proofWindowOpenBlock.Hash(),
		snapshot.SupplierOperatorAddress,
	)

	// Record the scheduled proof height for operators
	SetProofScheduledHeight(snapshot.SupplierOperatorAddress, snapshot.SessionID, float64(earliestProofHeight))

	logger.Info().
		Int64("earliest_proof_height", earliestProofHeight).
		Msg("waiting for assigned proof timing")

	// Wait for the proof path seed block (one before earliest proof height)
	proofPathSeedBlockHeight := earliestProofHeight - 1
	proofPathSeedBlock, err := lc.waitForBlock(ctx, proofPathSeedBlockHeight)
	if err != nil {
		return fmt.Errorf("failed to wait for proof path seed block: %w", err)
	}

	logger.Info().Msg("proof window timing reached - generating and submitting proof")

	// Generate the proof path from the seed block hash
	path := protocol.GetPathForProof(proofPathSeedBlock.Hash(), snapshot.SessionID)

	// Generate the proof
	proofBytes, err := lc.smstManager.ProveClosest(ctx, snapshot.SessionID, path)
	if err != nil {
		return fmt.Errorf("failed to generate proof: %w", err)
	}

	// Build the session header
	sessionHeader, err := lc.buildSessionHeader(ctx, snapshot)
	if err != nil {
		return fmt.Errorf("failed to build session header: %w", err)
	}

	// Calculate timeout height (proof window close)
	proofWindowClose := sharedtypes.GetProofWindowCloseHeight(sharedParams, snapshot.SessionEndHeight)

	// Build and submit the proof with retries
	proofMsg := &prooftypes.MsgSubmitProof{
		SupplierOperatorAddress: snapshot.SupplierOperatorAddress,
		SessionHeader:           sessionHeader,
		Proof:                   proofBytes,
	}

	var lastErr error
	for attempt := 1; attempt <= lc.config.ProofRetryAttempts; attempt++ {
		if err := lc.supplierClient.SubmitProofs(ctx, proofWindowClose, proofMsg); err != nil {
			lastErr = err
			logger.Warn().
				Err(err).
				Int(logging.FieldAttempt, attempt).
				Int(logging.FieldMaxRetry, lc.config.ProofRetryAttempts).
				Msg("proof submission failed, retrying")

			if attempt < lc.config.ProofRetryAttempts {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(lc.config.ProofRetryDelay):
					continue
				}
			}
		} else {
			// Success
			currentBlock := lc.blockClient.LastBlock(ctx)
			blocksAfterWindowOpen := float64(currentBlock.Height() - proofWindowOpenHeight)

			RecordProofSubmitted(snapshot.SupplierOperatorAddress)
			RecordProofSubmissionLatency(snapshot.SupplierOperatorAddress, blocksAfterWindowOpen)
			RecordComputeUnitsSettled(snapshot.SupplierOperatorAddress, snapshot.ServiceID, float64(snapshot.TotalComputeUnits))

			logger.Info().
				Int("proof_len", len(proofBytes)).
				Int64("blocks_after_window", int64(blocksAfterWindowOpen)).
				Msg("proof submitted successfully")

			return nil
		}
	}

	RecordProofError(snapshot.SupplierOperatorAddress, "exhausted_retries")
	return fmt.Errorf("proof submission failed after %d attempts: %w", lc.config.ProofRetryAttempts, lastErr)
}

// OnSessionSettled is called when a session is fully settled.
// It cleans up resources associated with the session.
func (lc *LifecycleCallback) OnSessionSettled(ctx context.Context, snapshot *SessionSnapshot) error {
	logger := lc.logger.With(
		logging.FieldSessionID, snapshot.SessionID,
	)

	logger.Info().
		Int64(logging.FieldCount, snapshot.RelayCount).
		Msg("session settled - cleaning up")

	// Record session settled metrics
	RecordSessionSettled(snapshot.SupplierOperatorAddress, snapshot.ServiceID)

	// Clean up session-specific metrics (gauges with session_id label)
	ClearSessionMetrics(snapshot.SupplierOperatorAddress, snapshot.SessionID, snapshot.ServiceID)

	// Clean up SMST
	if err := lc.smstManager.DeleteTree(ctx, snapshot.SessionID); err != nil {
		logger.Warn().Err(err).Msg("failed to delete SMST tree")
	}

	// Update snapshot manager
	if lc.snapshotMgr != nil {
		if err := lc.snapshotMgr.OnSessionSettled(ctx, snapshot.SessionID); err != nil {
			logger.Warn().Err(err).Msg("failed to update snapshot after settlement")
		}
	}

	// Remove session lock
	lc.removeSessionLock(snapshot.SessionID)

	return nil
}

// OnSessionExpired is called when a session expires without settling.
// This typically means the claim or proof window was missed.
func (lc *LifecycleCallback) OnSessionExpired(ctx context.Context, snapshot *SessionSnapshot, reason string) error {
	logger := lc.logger.With(
		logging.FieldSessionID, snapshot.SessionID,
	)

	logger.Warn().
		Str(logging.FieldReason, reason).
		Int64(logging.FieldCount, snapshot.RelayCount).
		Msg("session expired - rewards lost")

	// Record session failed metrics
	RecordSessionFailed(snapshot.SupplierOperatorAddress, snapshot.ServiceID, reason)

	// Clean up session-specific metrics (gauges with session_id label)
	ClearSessionMetrics(snapshot.SupplierOperatorAddress, snapshot.SessionID, snapshot.ServiceID)

	// Clean up SMST
	if err := lc.smstManager.DeleteTree(ctx, snapshot.SessionID); err != nil {
		logger.Warn().Err(err).Msg("failed to delete SMST tree")
	}

	// Remove session lock
	lc.removeSessionLock(snapshot.SessionID)

	return nil
}

// waitForBlock waits for a specific block height to be reached.
func (lc *LifecycleCallback) waitForBlock(ctx context.Context, targetHeight int64) (client.Block, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			block := lc.blockClient.LastBlock(ctx)
			if block.Height() >= targetHeight {
				return block, nil
			}

			lc.logger.Debug().
				Int64("current_height", block.Height()).
				Int64("target_height", targetHeight).
				Msg("waiting for block height")
		}
	}
}

// buildSessionHeader builds a session header from the snapshot.
// It queries the session from the blockchain to get complete information.
func (lc *LifecycleCallback) buildSessionHeader(ctx context.Context, snapshot *SessionSnapshot) (*sessiontypes.SessionHeader, error) {
	if lc.sessionClient != nil {
		// Query the session from the blockchain to get the complete header
		session, err := lc.sessionClient.GetSession(
			ctx,
			snapshot.ApplicationAddress,
			snapshot.ServiceID,
			snapshot.SessionStartHeight,
		)
		if err != nil {
			lc.logger.Warn().
				Err(err).
				Str(logging.FieldSessionID, snapshot.SessionID).
				Msg("failed to query session from blockchain, using snapshot data")
		} else if session != nil {
			return session.Header, nil
		}
	}

	// Fallback: build from snapshot data
	return &sessiontypes.SessionHeader{
		SessionId:               snapshot.SessionID,
		ApplicationAddress:      snapshot.ApplicationAddress,
		ServiceId:               snapshot.ServiceID,
		SessionStartBlockHeight: snapshot.SessionStartHeight,
		SessionEndBlockHeight:   snapshot.SessionEndHeight,
	}, nil
}

// Ensure LifecycleCallback implements SessionLifecycleCallback
var _ SessionLifecycleCallback = (*LifecycleCallback)(nil)
