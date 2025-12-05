package miner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// ClaimPipelineConfig contains configuration for the claim pipeline.
type ClaimPipelineConfig struct {
	// SupplierAddress is the supplier this pipeline is for.
	SupplierAddress string

	// MaxClaimsPerBatch is the maximum number of claims to submit in a single transaction.
	// Default: 10
	MaxClaimsPerBatch int

	// BatchWaitTime is how long to wait to accumulate claims before submitting.
	// Default: 5 seconds
	BatchWaitTime time.Duration

	// ClaimRetryAttempts is the number of times to retry failed claims.
	// Default: 3
	ClaimRetryAttempts int

	// ClaimRetryDelay is the delay between retry attempts.
	// Default: 1 second
	ClaimRetryDelay time.Duration
}

// DefaultClaimPipelineConfig returns sensible defaults.
func DefaultClaimPipelineConfig() ClaimPipelineConfig {
	return ClaimPipelineConfig{
		MaxClaimsPerBatch:  10,
		BatchWaitTime:      5 * time.Second,
		ClaimRetryAttempts: 3,
		ClaimRetryDelay:    1 * time.Second,
	}
}

// ClaimRequest represents a request to create a claim.
type ClaimRequest struct {
	// SessionID is the unique identifier for the session.
	SessionID string

	// SessionHeader contains the session metadata.
	SessionHeader *sessiontypes.SessionHeader

	// RootHash is the SMT root hash for the claim.
	RootHash []byte

	// SupplierOperatorAddress is the supplier submitting the claim.
	SupplierOperatorAddress string

	// SessionEndHeight is when the session ended.
	SessionEndHeight int64

	// Callback to invoke when claim is processed.
	Callback func(success bool, err error)
}

// ClaimResult represents the result of a claim submission.
type ClaimResult struct {
	// SessionID is the session that was claimed.
	SessionID string

	// Success indicates if the claim was submitted successfully.
	Success bool

	// Error contains any error that occurred.
	Error error

	// TxHash is the transaction hash if successful.
	TxHash string
}

// SMSTFlusher is the interface for flushing SMST trees to get root hashes.
type SMSTFlusher interface {
	// FlushTree flushes the SMST for a session and returns the root hash.
	FlushTree(ctx context.Context, sessionID string) (rootHash []byte, err error)

	// GetTreeRoot returns the root hash for an already-flushed session.
	GetTreeRoot(ctx context.Context, sessionID string) (rootHash []byte, err error)
}

// ClaimPipeline manages the claim submission process.
type ClaimPipeline struct {
	logger       polylog.Logger
	config       ClaimPipelineConfig
	txClient     client.SupplierClient
	sharedClient client.SharedQueryClient
	blockClient  client.BlockClient
	smstFlusher  SMSTFlusher

	// Claim queue
	claimQueue chan *ClaimRequest

	// Batching state
	pendingClaims   []*ClaimRequest
	pendingClaimsMu sync.Mutex
	batchTimer      *time.Timer

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	closed   bool
}

// NewClaimPipeline creates a new claim pipeline.
func NewClaimPipeline(
	logger polylog.Logger,
	txClient client.SupplierClient,
	sharedClient client.SharedQueryClient,
	blockClient client.BlockClient,
	smstFlusher SMSTFlusher,
	config ClaimPipelineConfig,
) *ClaimPipeline {
	if config.MaxClaimsPerBatch <= 0 {
		config.MaxClaimsPerBatch = 10
	}
	if config.BatchWaitTime <= 0 {
		config.BatchWaitTime = 5 * time.Second
	}
	if config.ClaimRetryAttempts <= 0 {
		config.ClaimRetryAttempts = 3
	}
	if config.ClaimRetryDelay <= 0 {
		config.ClaimRetryDelay = 1 * time.Second
	}

	return &ClaimPipeline{
		logger:        logging.ForSupplierComponent(logger, logging.ComponentClaimPipeline, config.SupplierAddress),
		config:        config,
		txClient:      txClient,
		sharedClient:  sharedClient,
		blockClient:   blockClient,
		smstFlusher:   smstFlusher,
		claimQueue:    make(chan *ClaimRequest, 1000),
		pendingClaims: make([]*ClaimRequest, 0, config.MaxClaimsPerBatch),
	}
}

// Start begins the claim pipeline workers.
func (p *ClaimPipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return fmt.Errorf("claim pipeline is closed")
	}

	p.ctx, p.cancelFn = context.WithCancel(ctx)
	p.mu.Unlock()

	// Start claim processor
	p.wg.Add(1)
	go p.claimProcessor(p.ctx)

	p.logger.Info().Msg("claim pipeline started")
	return nil
}

// SubmitClaim queues a claim for submission.
func (p *ClaimPipeline) SubmitClaim(ctx context.Context, req *ClaimRequest) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("claim pipeline is closed")
	}
	p.mu.RUnlock()

	select {
	case p.claimQueue <- req:
		p.logger.Debug().
			Str(logging.FieldSessionID, req.SessionID).
			Msg("claim queued")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CreateClaimFromSession creates a claim request from a session snapshot.
// This handles SMST flushing and claim construction.
func (p *ClaimPipeline) CreateClaimFromSession(
	ctx context.Context,
	snapshot *SessionSnapshot,
	sessionHeader *sessiontypes.SessionHeader,
) (*ClaimRequest, error) {
	// Flush the SMST to get the root hash
	rootHash, err := p.smstFlusher.FlushTree(ctx, snapshot.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to flush SMST: %w", err)
	}

	req := &ClaimRequest{
		SessionID:               snapshot.SessionID,
		SessionHeader:           sessionHeader,
		RootHash:                rootHash,
		SupplierOperatorAddress: snapshot.SupplierOperatorAddress,
		SessionEndHeight:        snapshot.SessionEndHeight,
	}

	return req, nil
}

// claimProcessor processes claims from the queue.
func (p *ClaimPipeline) claimProcessor(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Process any remaining claims before exiting
			p.flushPendingClaims(ctx)
			return

		case req := <-p.claimQueue:
			p.addToBatch(ctx, req)

		case <-p.getBatchTimerCh():
			p.flushPendingClaims(ctx)
		}
	}
}

// addToBatch adds a claim to the pending batch.
func (p *ClaimPipeline) addToBatch(ctx context.Context, req *ClaimRequest) {
	p.pendingClaimsMu.Lock()
	p.pendingClaims = append(p.pendingClaims, req)
	batchFull := len(p.pendingClaims) >= p.config.MaxClaimsPerBatch

	// Start timer if this is the first claim in the batch
	if len(p.pendingClaims) == 1 {
		p.startBatchTimer()
	}
	p.pendingClaimsMu.Unlock()

	// If batch is full, flush immediately
	if batchFull {
		p.flushPendingClaims(ctx)
	}
}

// startBatchTimer starts the batch timer.
func (p *ClaimPipeline) startBatchTimer() {
	if p.batchTimer != nil {
		p.batchTimer.Stop()
	}
	p.batchTimer = time.NewTimer(p.config.BatchWaitTime)
}

// getBatchTimerCh returns the batch timer channel.
func (p *ClaimPipeline) getBatchTimerCh() <-chan time.Time {
	p.pendingClaimsMu.Lock()
	defer p.pendingClaimsMu.Unlock()

	if p.batchTimer == nil {
		// Return a nil channel that never fires
		return nil
	}
	return p.batchTimer.C
}

// flushPendingClaims submits all pending claims.
func (p *ClaimPipeline) flushPendingClaims(ctx context.Context) {
	p.pendingClaimsMu.Lock()
	if len(p.pendingClaims) == 0 {
		p.pendingClaimsMu.Unlock()
		return
	}

	claims := p.pendingClaims
	p.pendingClaims = make([]*ClaimRequest, 0, p.config.MaxClaimsPerBatch)

	if p.batchTimer != nil {
		p.batchTimer.Stop()
		p.batchTimer = nil
	}
	p.pendingClaimsMu.Unlock()

	p.logger.Info().
		Int("count", len(claims)).
		Msg("flushing claim batch")

	// Submit the batch
	p.submitClaimBatch(ctx, claims)
}

// submitClaimBatch submits a batch of claims.
func (p *ClaimPipeline) submitClaimBatch(ctx context.Context, claims []*ClaimRequest) {
	if len(claims) == 0 {
		return
	}

	// Get shared params for timeout calculation
	sharedParams, err := p.sharedClient.GetParams(ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to get shared params for claim submission")
		p.notifyClaimResults(claims, false, err)
		return
	}

	// Group claims by session end height for proper timeout calculation
	claimsByEndHeight := make(map[int64][]*ClaimRequest)
	for _, claim := range claims {
		claimsByEndHeight[claim.SessionEndHeight] = append(claimsByEndHeight[claim.SessionEndHeight], claim)
	}

	for sessionEndHeight, heightClaims := range claimsByEndHeight {
		p.submitClaimsForHeight(ctx, heightClaims, sessionEndHeight, sharedParams)
	}
}

// submitClaimsForHeight submits claims for a specific session end height.
func (p *ClaimPipeline) submitClaimsForHeight(
	ctx context.Context,
	claims []*ClaimRequest,
	sessionEndHeight int64,
	sharedParams *sharedtypes.Params,
) {
	// Calculate timeout height (claim window close)
	claimWindowClose := sharedtypes.GetClaimWindowCloseHeight(sharedParams, sessionEndHeight)

	// Build claim messages
	claimMsgs := make([]client.MsgCreateClaim, 0, len(claims))
	for _, claim := range claims {
		msg := &prooftypes.MsgCreateClaim{
			SupplierOperatorAddress: claim.SupplierOperatorAddress,
			SessionHeader:           claim.SessionHeader,
			RootHash:                claim.RootHash,
		}
		claimMsgs = append(claimMsgs, msg)
	}

	p.logger.Info().
		Int("count", len(claimMsgs)).
		Int64("session_end_height", sessionEndHeight).
		Int64("timeout_height", claimWindowClose).
		Msg("submitting claims")

	// Submit with retries
	var lastErr error
	for attempt := 1; attempt <= p.config.ClaimRetryAttempts; attempt++ {
		err := p.txClient.CreateClaims(ctx, claimWindowClose, claimMsgs...)
		if err == nil {
			p.logger.Info().
				Int("count", len(claims)).
				Msg("claims submitted successfully")
			claimsSubmitted.WithLabelValues(p.config.SupplierAddress).Add(float64(len(claims)))
			p.notifyClaimResults(claims, true, nil)
			return
		}

		lastErr = err
		p.logger.Warn().
			Err(err).
			Int("attempt", attempt).
			Int("max_attempts", p.config.ClaimRetryAttempts).
			Msg("claim submission failed, retrying")

		if attempt < p.config.ClaimRetryAttempts {
			select {
			case <-ctx.Done():
				break
			case <-time.After(p.config.ClaimRetryDelay):
				// Continue to next attempt
			}
		}
	}

	p.logger.Error().
		Err(lastErr).
		Int("count", len(claims)).
		Msg("claim submission failed after all retries")
	claimErrors.WithLabelValues(p.config.SupplierAddress, "submission_failed").Add(float64(len(claims)))
	p.notifyClaimResults(claims, false, lastErr)
}

// notifyClaimResults notifies all claims of their result.
func (p *ClaimPipeline) notifyClaimResults(claims []*ClaimRequest, success bool, err error) {
	for _, claim := range claims {
		if claim.Callback != nil {
			claim.Callback(success, err)
		}
	}
}

// CalculateEarliestClaimHeight calculates when a supplier can start submitting claims.
// This uses the supplier's address and claim window open block hash for deterministic spreading.
func CalculateEarliestClaimHeight(
	sharedParams *sharedtypes.Params,
	sessionEndHeight int64,
	claimWindowOpenBlockHash []byte,
	supplierOperatorAddr string,
) int64 {
	return sharedtypes.GetEarliestSupplierClaimCommitHeight(
		sharedParams,
		sessionEndHeight,
		claimWindowOpenBlockHash,
		supplierOperatorAddr,
	)
}

// WaitForClaimWindow waits for the claim window to open and returns the block hash.
func (p *ClaimPipeline) WaitForClaimWindow(
	ctx context.Context,
	sessionEndHeight int64,
) (claimWindowOpenHeight int64, blockHash []byte, err error) {
	sharedParams, err := p.sharedClient.GetParams(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get shared params: %w", err)
	}

	claimWindowOpen := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)

	p.logger.Info().
		Int64("session_end_height", sessionEndHeight).
		Int64("claim_window_open", claimWindowOpen).
		Msg("waiting for claim window to open")

	// Poll for the block height
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		case <-ticker.C:
			block := p.blockClient.LastBlock(ctx)
			currentHeight := block.Height()

			if currentHeight >= claimWindowOpen {
				p.logger.Info().
					Int64("current_height", currentHeight).
					Int64("claim_window_open", claimWindowOpen).
					Msg("claim window is open")
				return claimWindowOpen, block.Hash(), nil
			}
		}
	}
}

// Close gracefully shuts down the claim pipeline.
func (p *ClaimPipeline) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	if p.cancelFn != nil {
		p.cancelFn()
	}

	p.wg.Wait()

	p.logger.Info().Msg("claim pipeline closed")
	return nil
}

// ClaimBatchResult represents the result of a batch of claim submissions.
type ClaimBatchResult struct {
	// SuccessfulClaims are the claims that were submitted successfully.
	SuccessfulClaims []*ClaimRequest

	// FailedClaims are the claims that failed to submit.
	FailedClaims []*ClaimRequest

	// TxHash is the transaction hash if any claims were submitted.
	TxHash string

	// Error contains any error from the batch submission.
	Error error
}

// ClaimBatcher provides batch claim submission.
type ClaimBatcher struct {
	logger    polylog.Logger
	txClient  client.SupplierClient
	supplier  string
	batchSize int
}

// NewClaimBatcher creates a new claim batcher.
func NewClaimBatcher(
	logger polylog.Logger,
	txClient client.SupplierClient,
	supplier string,
	batchSize int,
) *ClaimBatcher {
	if batchSize <= 0 {
		batchSize = 10
	}

	return &ClaimBatcher{
		logger:    logging.ForSupplierComponent(logger, logging.ComponentClaimBatcher, supplier),
		txClient:  txClient,
		supplier:  supplier,
		batchSize: batchSize,
	}
}

// SubmitBatch submits a batch of claims and returns results.
func (b *ClaimBatcher) SubmitBatch(
	ctx context.Context,
	claims []*ClaimRequest,
	timeoutHeight int64,
) *ClaimBatchResult {
	result := &ClaimBatchResult{
		SuccessfulClaims: make([]*ClaimRequest, 0),
		FailedClaims:     make([]*ClaimRequest, 0),
	}

	if len(claims) == 0 {
		return result
	}

	// Split into batches
	for i := 0; i < len(claims); i += b.batchSize {
		end := i + b.batchSize
		if end > len(claims) {
			end = len(claims)
		}
		batch := claims[i:end]

		batchResult := b.submitSingleBatch(ctx, batch, timeoutHeight)
		result.SuccessfulClaims = append(result.SuccessfulClaims, batchResult.SuccessfulClaims...)
		result.FailedClaims = append(result.FailedClaims, batchResult.FailedClaims...)

		if batchResult.Error != nil {
			result.Error = batchResult.Error
		}
	}

	return result
}

// submitSingleBatch submits a single batch of claims.
func (b *ClaimBatcher) submitSingleBatch(
	ctx context.Context,
	claims []*ClaimRequest,
	timeoutHeight int64,
) *ClaimBatchResult {
	result := &ClaimBatchResult{}

	claimMsgs := make([]client.MsgCreateClaim, 0, len(claims))
	for _, claim := range claims {
		msg := &prooftypes.MsgCreateClaim{
			SupplierOperatorAddress: claim.SupplierOperatorAddress,
			SessionHeader:           claim.SessionHeader,
			RootHash:                claim.RootHash,
		}
		claimMsgs = append(claimMsgs, msg)
	}

	err := b.txClient.CreateClaims(ctx, timeoutHeight, claimMsgs...)
	if err != nil {
		result.Error = err
		result.FailedClaims = claims
		return result
	}

	result.SuccessfulClaims = claims
	return result
}
