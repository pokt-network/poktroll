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

// ProofPipelineConfig contains configuration for the proof pipeline.
type ProofPipelineConfig struct {
	// SupplierAddress is the supplier this pipeline is for.
	SupplierAddress string

	// MaxProofsPerBatch is the maximum number of proofs to submit in a single transaction.
	MaxProofsPerBatch int

	// BatchWaitTime is how long to wait to accumulate proofs before submitting.
	BatchWaitTime time.Duration

	// ProofRetryAttempts is the number of times to retry failed proofs.
	ProofRetryAttempts int

	// ProofRetryDelay is the delay between retry attempts.
	ProofRetryDelay time.Duration
}

// DefaultProofPipelineConfig returns sensible defaults.
func DefaultProofPipelineConfig() ProofPipelineConfig {
	return ProofPipelineConfig{
		MaxProofsPerBatch:  10,
		BatchWaitTime:      5 * time.Second,
		ProofRetryAttempts: 3,
		ProofRetryDelay:    1 * time.Second,
	}
}

// ProofRequest represents a request to submit a proof.
type ProofRequest struct {
	// SessionID is the unique identifier for the session.
	SessionID string

	// SessionHeader contains the session metadata.
	SessionHeader *sessiontypes.SessionHeader

	// ProofBytes is the serialized proof.
	ProofBytes []byte

	// SupplierOperatorAddress is the supplier submitting the proof.
	SupplierOperatorAddress string

	// SessionEndHeight is when the session ended.
	SessionEndHeight int64

	// Callback to invoke when proof is processed.
	Callback func(success bool, err error)
}

// ProofResult represents the result of a proof submission.
type ProofResult struct {
	SessionID string
	Success   bool
	Error     error
	TxHash    string
}

// SMSTProver provides proof generation from SMST trees.
type SMSTProver interface {
	// ProveClosest generates a proof for the closest leaf to the given path.
	ProveClosest(ctx context.Context, sessionID string, path []byte) (proofBytes []byte, err error)

	// GetClaimRoot returns the root hash for a session.
	GetClaimRoot(ctx context.Context, sessionID string) (rootHash []byte, err error)
}

// ProofRequirementChecker determines if a proof is required for a claim.
type ProofRequirementChecker interface {
	// IsProofRequired determines if a proof is required for the given claim.
	IsProofRequired(
		ctx context.Context,
		sessionID string,
		claimRootHash []byte,
		proofRequirementSeedBlockHash []byte,
	) (bool, error)
}

// ProofPipeline manages the proof submission process.
type ProofPipeline struct {
	logger             polylog.Logger
	config             ProofPipelineConfig
	txClient           client.SupplierClient
	sharedClient       client.SharedQueryClient
	proofClient        client.ProofQueryClient
	blockClient        client.BlockClient
	smstProver         SMSTProver
	requirementChecker ProofRequirementChecker

	// Proof queue
	proofQueue chan *ProofRequest

	// Batching state
	pendingProofs   []*ProofRequest
	pendingProofsMu sync.Mutex
	batchTimer      *time.Timer

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	closed   bool
}

// NewProofPipeline creates a new proof pipeline.
func NewProofPipeline(
	logger polylog.Logger,
	txClient client.SupplierClient,
	sharedClient client.SharedQueryClient,
	proofClient client.ProofQueryClient,
	blockClient client.BlockClient,
	smstProver SMSTProver,
	config ProofPipelineConfig,
) *ProofPipeline {
	if config.MaxProofsPerBatch <= 0 {
		config.MaxProofsPerBatch = 10
	}
	if config.BatchWaitTime <= 0 {
		config.BatchWaitTime = 5 * time.Second
	}
	if config.ProofRetryAttempts <= 0 {
		config.ProofRetryAttempts = 3
	}
	if config.ProofRetryDelay <= 0 {
		config.ProofRetryDelay = 1 * time.Second
	}

	return &ProofPipeline{
		logger:        logging.ForSupplierComponent(logger, logging.ComponentProofPipeline, config.SupplierAddress),
		config:        config,
		txClient:      txClient,
		sharedClient:  sharedClient,
		proofClient:   proofClient,
		blockClient:   blockClient,
		smstProver:    smstProver,
		proofQueue:    make(chan *ProofRequest, 1000),
		pendingProofs: make([]*ProofRequest, 0, config.MaxProofsPerBatch),
	}
}

// Start begins the proof pipeline workers.
func (p *ProofPipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return fmt.Errorf("proof pipeline is closed")
	}

	p.ctx, p.cancelFn = context.WithCancel(ctx)
	p.mu.Unlock()

	// Start proof processor
	p.wg.Add(1)
	go p.proofProcessor(p.ctx)

	p.logger.Info().Msg("proof pipeline started")
	return nil
}

// SubmitProof queues a proof for submission.
func (p *ProofPipeline) SubmitProof(ctx context.Context, req *ProofRequest) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("proof pipeline is closed")
	}
	p.mu.RUnlock()

	select {
	case p.proofQueue <- req:
		p.logger.Debug().
			Str(logging.FieldSessionID, req.SessionID).
			Msg("proof queued")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GenerateProofForSession generates a proof for a session that has been claimed.
func (p *ProofPipeline) GenerateProofForSession(
	ctx context.Context,
	snapshot *SessionSnapshot,
	sessionHeader *sessiontypes.SessionHeader,
	proofPathSeedBlockHash []byte,
) (*ProofRequest, error) {
	// Calculate the proof path from the seed block hash and session ID
	path := protocol.GetPathForProof(proofPathSeedBlockHash, snapshot.SessionID)

	// Generate the proof
	proofBytes, err := p.smstProver.ProveClosest(ctx, snapshot.SessionID, path)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	req := &ProofRequest{
		SessionID:               snapshot.SessionID,
		SessionHeader:           sessionHeader,
		ProofBytes:              proofBytes,
		SupplierOperatorAddress: snapshot.SupplierOperatorAddress,
		SessionEndHeight:        snapshot.SessionEndHeight,
	}

	return req, nil
}

// CheckProofRequired determines if a proof is required for the given session.
func (p *ProofPipeline) CheckProofRequired(
	ctx context.Context,
	sessionID string,
	claimRootHash []byte,
	proofRequirementSeedBlockHash []byte,
) (bool, error) {
	if p.requirementChecker != nil {
		return p.requirementChecker.IsProofRequired(
			ctx,
			sessionID,
			claimRootHash,
			proofRequirementSeedBlockHash,
		)
	}

	// Default implementation: always require proof
	// In production, this should use the proof module params
	return true, nil
}

// proofProcessor processes proofs from the queue.
func (p *ProofPipeline) proofProcessor(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			p.flushPendingProofs(ctx)
			return

		case req := <-p.proofQueue:
			p.addToBatch(ctx, req)

		case <-p.getBatchTimerCh():
			p.flushPendingProofs(ctx)
		}
	}
}

// addToBatch adds a proof to the pending batch.
func (p *ProofPipeline) addToBatch(ctx context.Context, req *ProofRequest) {
	p.pendingProofsMu.Lock()
	p.pendingProofs = append(p.pendingProofs, req)
	batchFull := len(p.pendingProofs) >= p.config.MaxProofsPerBatch

	if len(p.pendingProofs) == 1 {
		p.startBatchTimer()
	}
	p.pendingProofsMu.Unlock()

	if batchFull {
		p.flushPendingProofs(ctx)
	}
}

// startBatchTimer starts the batch timer.
func (p *ProofPipeline) startBatchTimer() {
	if p.batchTimer != nil {
		p.batchTimer.Stop()
	}
	p.batchTimer = time.NewTimer(p.config.BatchWaitTime)
}

// getBatchTimerCh returns the batch timer channel.
func (p *ProofPipeline) getBatchTimerCh() <-chan time.Time {
	p.pendingProofsMu.Lock()
	defer p.pendingProofsMu.Unlock()

	if p.batchTimer == nil {
		return nil
	}
	return p.batchTimer.C
}

// flushPendingProofs submits all pending proofs.
func (p *ProofPipeline) flushPendingProofs(ctx context.Context) {
	p.pendingProofsMu.Lock()
	if len(p.pendingProofs) == 0 {
		p.pendingProofsMu.Unlock()
		return
	}

	proofs := p.pendingProofs
	p.pendingProofs = make([]*ProofRequest, 0, p.config.MaxProofsPerBatch)

	if p.batchTimer != nil {
		p.batchTimer.Stop()
		p.batchTimer = nil
	}
	p.pendingProofsMu.Unlock()

	p.logger.Info().
		Int("count", len(proofs)).
		Msg("flushing proof batch")

	p.submitProofBatch(ctx, proofs)
}

// submitProofBatch submits a batch of proofs.
func (p *ProofPipeline) submitProofBatch(ctx context.Context, proofs []*ProofRequest) {
	if len(proofs) == 0 {
		return
	}

	sharedParams, err := p.sharedClient.GetParams(ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("failed to get shared params for proof submission")
		p.notifyProofResults(proofs, false, err)
		return
	}

	// Group proofs by session end height
	proofsByEndHeight := make(map[int64][]*ProofRequest)
	for _, proof := range proofs {
		proofsByEndHeight[proof.SessionEndHeight] = append(proofsByEndHeight[proof.SessionEndHeight], proof)
	}

	for sessionEndHeight, heightProofs := range proofsByEndHeight {
		p.submitProofsForHeight(ctx, heightProofs, sessionEndHeight, sharedParams)
	}
}

// submitProofsForHeight submits proofs for a specific session end height.
func (p *ProofPipeline) submitProofsForHeight(
	ctx context.Context,
	proofs []*ProofRequest,
	sessionEndHeight int64,
	sharedParams *sharedtypes.Params,
) {
	proofWindowClose := sharedtypes.GetProofWindowCloseHeight(sharedParams, sessionEndHeight)

	proofMsgs := make([]client.MsgSubmitProof, 0, len(proofs))
	for _, proof := range proofs {
		msg := &prooftypes.MsgSubmitProof{
			SupplierOperatorAddress: proof.SupplierOperatorAddress,
			SessionHeader:           proof.SessionHeader,
			Proof:                   proof.ProofBytes,
		}
		proofMsgs = append(proofMsgs, msg)
	}

	p.logger.Info().
		Int("count", len(proofMsgs)).
		Int64("session_end_height", sessionEndHeight).
		Int64("timeout_height", proofWindowClose).
		Msg("submitting proofs")

	var lastErr error
	for attempt := 1; attempt <= p.config.ProofRetryAttempts; attempt++ {
		err := p.txClient.SubmitProofs(ctx, proofWindowClose, proofMsgs...)
		if err == nil {
			p.logger.Info().
				Int("count", len(proofs)).
				Msg("proofs submitted successfully")
			proofsSubmitted.WithLabelValues(p.config.SupplierAddress).Add(float64(len(proofs)))
			p.notifyProofResults(proofs, true, nil)
			return
		}

		lastErr = err
		p.logger.Warn().
			Err(err).
			Int("attempt", attempt).
			Int("max_attempts", p.config.ProofRetryAttempts).
			Msg("proof submission failed, retrying")

		if attempt < p.config.ProofRetryAttempts {
			select {
			case <-ctx.Done():
				break
			case <-time.After(p.config.ProofRetryDelay):
			}
		}
	}

	p.logger.Error().
		Err(lastErr).
		Int("count", len(proofs)).
		Msg("proof submission failed after all retries")
	proofErrors.WithLabelValues(p.config.SupplierAddress, "submission_failed").Add(float64(len(proofs)))
	p.notifyProofResults(proofs, false, lastErr)
}

// notifyProofResults notifies all proofs of their result.
func (p *ProofPipeline) notifyProofResults(proofs []*ProofRequest, success bool, err error) {
	for _, proof := range proofs {
		if proof.Callback != nil {
			proof.Callback(success, err)
		}
	}
}

// WaitForProofWindow waits for the proof window to open.
func (p *ProofPipeline) WaitForProofWindow(
	ctx context.Context,
	sessionEndHeight int64,
) (proofWindowOpenHeight int64, blockHash []byte, err error) {
	sharedParams, err := p.sharedClient.GetParams(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get shared params: %w", err)
	}

	proofWindowOpen := sharedtypes.GetProofWindowOpenHeight(sharedParams, sessionEndHeight)

	p.logger.Info().
		Int64("session_end_height", sessionEndHeight).
		Int64("proof_window_open", proofWindowOpen).
		Msg("waiting for proof window to open")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return 0, nil, ctx.Err()
		case <-ticker.C:
			block := p.blockClient.LastBlock(ctx)
			currentHeight := block.Height()

			if currentHeight >= proofWindowOpen {
				p.logger.Info().
					Int64("current_height", currentHeight).
					Int64("proof_window_open", proofWindowOpen).
					Msg("proof window is open")
				return proofWindowOpen, block.Hash(), nil
			}
		}
	}
}

// CalculateEarliestProofHeight calculates when a supplier can start submitting proofs.
func CalculateEarliestProofHeight(
	sharedParams *sharedtypes.Params,
	sessionEndHeight int64,
	proofWindowOpenBlockHash []byte,
	supplierOperatorAddr string,
) int64 {
	return sharedtypes.GetEarliestSupplierProofCommitHeight(
		sharedParams,
		sessionEndHeight,
		proofWindowOpenBlockHash,
		supplierOperatorAddr,
	)
}

// Close gracefully shuts down the proof pipeline.
func (p *ProofPipeline) Close() error {
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

	p.logger.Info().Msg("proof pipeline closed")
	return nil
}

// ProofBatcher provides batch proof submission.
type ProofBatcher struct {
	logger    polylog.Logger
	txClient  client.SupplierClient
	supplier  string
	batchSize int
}

// NewProofBatcher creates a new proof batcher.
func NewProofBatcher(
	logger polylog.Logger,
	txClient client.SupplierClient,
	supplier string,
	batchSize int,
) *ProofBatcher {
	if batchSize <= 0 {
		batchSize = 10
	}

	return &ProofBatcher{
		logger:    logging.ForSupplierComponent(logger, logging.ComponentProofBatcher, supplier),
		txClient:  txClient,
		supplier:  supplier,
		batchSize: batchSize,
	}
}

// ProofBatchResult represents the result of a batch proof submission.
type ProofBatchResult struct {
	SuccessfulProofs []*ProofRequest
	FailedProofs     []*ProofRequest
	TxHash           string
	Error            error
}

// SubmitBatch submits a batch of proofs and returns results.
func (b *ProofBatcher) SubmitBatch(
	ctx context.Context,
	proofs []*ProofRequest,
	timeoutHeight int64,
) *ProofBatchResult {
	result := &ProofBatchResult{
		SuccessfulProofs: make([]*ProofRequest, 0),
		FailedProofs:     make([]*ProofRequest, 0),
	}

	if len(proofs) == 0 {
		return result
	}

	for i := 0; i < len(proofs); i += b.batchSize {
		end := i + b.batchSize
		if end > len(proofs) {
			end = len(proofs)
		}
		batch := proofs[i:end]

		batchResult := b.submitSingleBatch(ctx, batch, timeoutHeight)
		result.SuccessfulProofs = append(result.SuccessfulProofs, batchResult.SuccessfulProofs...)
		result.FailedProofs = append(result.FailedProofs, batchResult.FailedProofs...)

		if batchResult.Error != nil {
			result.Error = batchResult.Error
		}
	}

	return result
}

// submitSingleBatch submits a single batch of proofs.
func (b *ProofBatcher) submitSingleBatch(
	ctx context.Context,
	proofs []*ProofRequest,
	timeoutHeight int64,
) *ProofBatchResult {
	result := &ProofBatchResult{}

	proofMsgs := make([]client.MsgSubmitProof, 0, len(proofs))
	for _, proof := range proofs {
		msg := &prooftypes.MsgSubmitProof{
			SupplierOperatorAddress: proof.SupplierOperatorAddress,
			SessionHeader:           proof.SessionHeader,
			Proof:                   proof.ProofBytes,
		}
		proofMsgs = append(proofMsgs, msg)
	}

	err := b.txClient.SubmitProofs(ctx, timeoutHeight, proofMsgs...)
	if err != nil {
		result.Error = err
		result.FailedProofs = proofs
		return result
	}

	result.SuccessfulProofs = proofs
	return result
}

// DefaultProofRequirementChecker implements basic proof requirement checking.
type DefaultProofRequirementChecker struct {
	logger       polylog.Logger
	proofClient  client.ProofQueryClient
	sharedClient client.SharedQueryClient
}

// NewDefaultProofRequirementChecker creates a new proof requirement checker.
func NewDefaultProofRequirementChecker(
	logger polylog.Logger,
	proofClient client.ProofQueryClient,
	sharedClient client.SharedQueryClient,
) *DefaultProofRequirementChecker {
	return &DefaultProofRequirementChecker{
		logger:       logging.ForComponent(logger, logging.ComponentProofChecker),
		proofClient:  proofClient,
		sharedClient: sharedClient,
	}
}
