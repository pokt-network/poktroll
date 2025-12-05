package relayer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/ha/cache"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	// ValidationModeLazy trusts requests and validates asynchronously.
	// Lower latency but may process invalid relays.
	ValidationModeLazy ValidationMode = "lazy"

	// ValidationModeSampled validates a percentage of requests synchronously.
	// Balance between latency and security.
	ValidationModeSampled ValidationMode = "sampled"
)

// SessionValidatorConfig contains configuration for the session validator.
type SessionValidatorConfig struct {
	// Mode determines how validation is performed.
	Mode ValidationMode

	// SampleRate is the percentage of requests to validate synchronously (0.0-1.0).
	// Only used when Mode is ValidationModeSampled.
	SampleRate float64

	// AsyncQueueSize is the buffer size for async validation queue.
	AsyncQueueSize int

	// ValidationTimeout is the timeout for each validation operation.
	ValidationTimeout time.Duration
}

// DefaultSessionValidatorConfig returns sensible defaults.
func DefaultSessionValidatorConfig() SessionValidatorConfig {
	return SessionValidatorConfig{
		Mode:              ValidationModeEager,
		SampleRate:        0.1, // 10% sample rate
		AsyncQueueSize:    10000,
		ValidationTimeout: 5 * time.Second,
	}
}

// SessionValidationRequest contains data needed for session validation.
type SessionValidationRequest struct {
	AppAddress         string
	ServiceID          string
	SessionID          string
	SessionEndHeight   int64
	SupplierOperator   string
	RelayRequest       *servicetypes.RelayRequest
	CurrentBlockHeight int64
}

// SessionValidationResult contains the outcome of validation.
type SessionValidationResult struct {
	IsValid       bool
	FailureReason string
	Session       *cache.SessionValidationResult
}

// SessionValidator validates incoming relay requests.
type SessionValidator struct {
	logger       polylog.Logger
	config       SessionValidatorConfig
	sessionCache cache.SessionCache
	sharedClient client.SharedQueryClient
	blockClient  client.BlockClient
	ringClient   crypto.RingClient

	// Known supplier operator addresses
	supplierAddrs   map[string]bool
	supplierAddrsMu sync.RWMutex

	// Async validation channel
	asyncValidationCh chan *SessionValidationRequest

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
	closed   bool
	mu       sync.RWMutex
}

// NewSessionValidator creates a new session validator.
func NewSessionValidator(
	logger polylog.Logger,
	sessionCache cache.SessionCache,
	sharedClient client.SharedQueryClient,
	blockClient client.BlockClient,
	ringClient crypto.RingClient,
	config SessionValidatorConfig,
) *SessionValidator {
	if config.AsyncQueueSize == 0 {
		config.AsyncQueueSize = 10000
	}
	if config.ValidationTimeout == 0 {
		config.ValidationTimeout = 5 * time.Second
	}

	return &SessionValidator{
		logger:            logging.ForComponent(logger, logging.ComponentSessionValidator),
		config:            config,
		sessionCache:      sessionCache,
		sharedClient:      sharedClient,
		blockClient:       blockClient,
		ringClient:        ringClient,
		supplierAddrs:     make(map[string]bool),
		asyncValidationCh: make(chan *SessionValidationRequest, config.AsyncQueueSize),
	}
}

// SetSupplierAddresses updates the list of known supplier addresses.
func (sv *SessionValidator) SetSupplierAddresses(addresses []string) {
	sv.supplierAddrsMu.Lock()
	defer sv.supplierAddrsMu.Unlock()

	sv.supplierAddrs = make(map[string]bool, len(addresses))
	for _, addr := range addresses {
		sv.supplierAddrs[addr] = true
	}

	sv.logger.Info().
		Int("count", len(addresses)).
		Msg("updated supplier addresses")
}

// IsKnownSupplier checks if the address belongs to a supplier we serve.
func (sv *SessionValidator) IsKnownSupplier(addr string) bool {
	sv.supplierAddrsMu.RLock()
	defer sv.supplierAddrsMu.RUnlock()
	return sv.supplierAddrs[addr]
}

// Start begins the async validation workers.
func (sv *SessionValidator) Start(ctx context.Context) error {
	sv.mu.Lock()
	if sv.closed {
		sv.mu.Unlock()
		return fmt.Errorf("validator is closed")
	}
	sv.ctx, sv.cancelFn = context.WithCancel(ctx)
	sv.mu.Unlock()

	// Start async validation workers
	numWorkers := 4 // TODO: make configurable
	for i := 0; i < numWorkers; i++ {
		sv.wg.Add(1)
		go sv.asyncValidationWorker(sv.ctx)
	}

	sv.logger.Info().
		Int("workers", numWorkers).
		Str("mode", sv.modeString()).
		Msg("session validator started")

	return nil
}

func (sv *SessionValidator) modeString() string {
	switch sv.config.Mode {
	case ValidationModeEager:
		return "eager"
	case ValidationModeLazy:
		return "lazy"
	case ValidationModeSampled:
		return fmt.Sprintf("sampled (%.0f%%)", sv.config.SampleRate*100)
	default:
		return "unknown"
	}
}

// ValidateRelayRequest validates a relay request based on the configured mode.
func (sv *SessionValidator) ValidateRelayRequest(
	ctx context.Context,
	req *SessionValidationRequest,
) (*SessionValidationResult, error) {
	sv.mu.RLock()
	if sv.closed {
		sv.mu.RUnlock()
		return nil, fmt.Errorf("validator is closed")
	}
	sv.mu.RUnlock()

	switch sv.config.Mode {
	case ValidationModeEager:
		return sv.validateSync(ctx, req)

	case ValidationModeLazy:
		// Queue for async validation and return success
		sv.queueAsyncValidation(req)
		return &SessionValidationResult{IsValid: true}, nil

	case ValidationModeSampled:
		// Validate sample synchronously, rest async
		if sv.shouldSample() {
			return sv.validateSync(ctx, req)
		}
		sv.queueAsyncValidation(req)
		return &SessionValidationResult{IsValid: true}, nil

	default:
		return sv.validateSync(ctx, req)
	}
}

// validateSync performs synchronous validation of a relay request.
func (sv *SessionValidator) validateSync(
	ctx context.Context,
	req *SessionValidationRequest,
) (*SessionValidationResult, error) {
	startTime := time.Now()
	defer func() {
		sessionValidationLatency.WithLabelValues("sync").Observe(time.Since(startTime).Seconds())
	}()

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, sv.config.ValidationTimeout)
	defer cancel()

	// Step 1: Check if supplier is known to us
	if !sv.IsKnownSupplier(req.SupplierOperator) {
		sessionValidationsTotal.WithLabelValues("invalid", "unknown_supplier").Inc()
		return &SessionValidationResult{
			IsValid:       false,
			FailureReason: fmt.Sprintf("unknown supplier: %s", req.SupplierOperator),
		}, nil
	}

	// Step 2: Check session rewardability
	if !sv.sessionCache.IsSessionRewardable(ctx, req.SessionID) {
		sessionValidationsTotal.WithLabelValues("invalid", "not_rewardable").Inc()
		return &SessionValidationResult{
			IsValid:       false,
			FailureReason: "session is no longer rewardable",
		}, nil
	}

	// Step 3: Check if session is within valid time window
	targetHeight, err := sv.getTargetSessionBlockHeight(ctx, req)
	if err != nil {
		sessionValidationsTotal.WithLabelValues("error", "height_check").Inc()
		return nil, fmt.Errorf("failed to get target session height: %w", err)
	}

	// Step 4: Verify relay request signature (if ring client is available)
	if sv.ringClient != nil && req.RelayRequest != nil {
		if sigErr := sv.ringClient.VerifyRelayRequestSignature(ctx, req.RelayRequest); sigErr != nil {
			sessionValidationsTotal.WithLabelValues("invalid", "bad_signature").Inc()
			return &SessionValidationResult{
				IsValid:       false,
				FailureReason: fmt.Sprintf("invalid signature: %v", sigErr),
			}, nil
		}
	}

	// Step 5: Verify session exists and matches
	session, err := sv.sessionCache.GetSession(ctx, req.AppAddress, req.ServiceID, targetHeight)
	if err != nil {
		sessionValidationsTotal.WithLabelValues("error", "session_query").Inc()
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.SessionId != req.SessionID {
		sessionValidationsTotal.WithLabelValues("invalid", "session_mismatch").Inc()
		return &SessionValidationResult{
			IsValid:       false,
			FailureReason: fmt.Sprintf("session ID mismatch: expected %s, got %s", session.SessionId, req.SessionID),
		}, nil
	}

	// Step 6: Verify supplier is in session
	supplierFound := false
	for _, supplier := range session.Suppliers {
		if supplier.OperatorAddress == req.SupplierOperator {
			supplierFound = true
			break
		}
	}
	if !supplierFound {
		sessionValidationsTotal.WithLabelValues("invalid", "supplier_not_in_session").Inc()
		return &SessionValidationResult{
			IsValid:       false,
			FailureReason: "supplier not in session",
		}, nil
	}

	sessionValidationsTotal.WithLabelValues("valid", "").Inc()
	return &SessionValidationResult{
		IsValid: true,
		Session: &cache.SessionValidationResult{
			AppAddress:       req.AppAddress,
			ServiceId:        req.ServiceID,
			BlockHeight:      targetHeight,
			SessionID:        req.SessionID,
			SessionEndHeight: session.Header.SessionEndBlockHeight,
			IsValid:          true,
			ValidatedAt:      time.Now().Unix(),
		},
	}, nil
}

// getTargetSessionBlockHeight determines the block height to use for session lookup.
func (sv *SessionValidator) getTargetSessionBlockHeight(
	ctx context.Context,
	req *SessionValidationRequest,
) (int64, error) {
	currentBlock := sv.blockClient.LastBlock(ctx)
	currentHeight := currentBlock.Height()

	// If the request has a session end height that's in the past, check grace period
	if req.SessionEndHeight < currentHeight {
		sharedParams, err := sv.sharedClient.GetParams(ctx)
		if err != nil {
			return 0, err
		}

		// Check if still within grace period
		if !sharedtypes.IsGracePeriodElapsed(sharedParams, req.SessionEndHeight, currentHeight) {
			return req.SessionEndHeight, nil
		}

		// Session has fully expired
		return 0, fmt.Errorf("session expired: end height %d, current %d", req.SessionEndHeight, currentHeight)
	}

	return currentHeight, nil
}

// CheckRewardEligibility verifies the relay's session hasn't expired for reward purposes.
func (sv *SessionValidator) CheckRewardEligibility(
	ctx context.Context,
	sessionEndHeight int64,
) (bool, error) {
	currentBlock := sv.blockClient.LastBlock(ctx)
	currentHeight := currentBlock.Height()

	sharedParams, err := sv.sharedClient.GetParams(ctx)
	if err != nil {
		return false, err
	}

	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)

	// If current height is >= claim window opening, session is no longer rewardable
	return currentHeight < claimWindowOpenHeight, nil
}

// queueAsyncValidation adds a validation request to the async queue.
func (sv *SessionValidator) queueAsyncValidation(req *SessionValidationRequest) {
	select {
	case sv.asyncValidationCh <- req:
		asyncValidationQueued.Inc()
	default:
		// Queue full - drop the request
		asyncValidationDropped.Inc()
		sv.logger.Warn().
			Str(logging.FieldSessionID, req.SessionID).
			Msg("async validation queue full, dropping request")
	}
}

// asyncValidationWorker processes async validation requests.
func (sv *SessionValidator) asyncValidationWorker(ctx context.Context) {
	defer sv.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-sv.asyncValidationCh:
			if !ok {
				return
			}
			sv.processAsyncValidation(ctx, req)
		}
	}
}

// processAsyncValidation handles a single async validation request.
func (sv *SessionValidator) processAsyncValidation(ctx context.Context, req *SessionValidationRequest) {
	startTime := time.Now()
	defer func() {
		sessionValidationLatency.WithLabelValues("async").Observe(time.Since(startTime).Seconds())
	}()

	result, err := sv.validateSync(ctx, req)
	if err != nil {
		sv.logger.Warn().
			Err(err).
			Str(logging.FieldSessionID, req.SessionID).
			Msg("async validation error")
		return
	}

	if !result.IsValid {
		// Mark session as non-rewardable if validation fails
		if err := sv.sessionCache.MarkSessionNonRewardable(ctx, req.SessionID, result.FailureReason); err != nil {
			sv.logger.Warn().
				Err(err).
				Str(logging.FieldSessionID, req.SessionID).
				Msg("failed to mark session non-rewardable")
		}

		sv.logger.Warn().
			Str(logging.FieldSessionID, req.SessionID).
			Str(logging.FieldReason, result.FailureReason).
			Msg("async validation failed - session marked non-rewardable")
	}
}

// shouldSample returns true if this request should be sampled for sync validation.
func (sv *SessionValidator) shouldSample() bool {
	// Simple random sampling
	// In production, could use more sophisticated approaches
	return time.Now().UnixNano()%100 < int64(sv.config.SampleRate*100)
}

// Close gracefully shuts down the validator.
func (sv *SessionValidator) Close() error {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	if sv.closed {
		return nil
	}
	sv.closed = true

	if sv.cancelFn != nil {
		sv.cancelFn()
	}

	close(sv.asyncValidationCh)
	sv.wg.Wait()

	sv.logger.Info().Msg("session validator closed")
	return nil
}
