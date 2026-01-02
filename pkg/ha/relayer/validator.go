package relayer

import (
	"context"
	"fmt"
	"sync"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/ha/cache"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// RelayValidator is responsible for validating relay requests.
// It verifies ring signatures and session validity using cached data
// to minimize on-chain queries.
type RelayValidator interface {
	// ValidateRelayRequest validates a relay request.
	// Returns nil if the request is valid, or an error describing the validation failure.
	ValidateRelayRequest(ctx context.Context, relayRequest *servicetypes.RelayRequest) error

	// CheckRewardEligibility checks if a relay is still eligible for rewards.
	// Returns nil if eligible, or an error if the relay is past the claim window.
	CheckRewardEligibility(ctx context.Context, relayRequest *servicetypes.RelayRequest) error

	// GetCurrentBlockHeight returns the current block height used for validation.
	GetCurrentBlockHeight() int64

	// SetCurrentBlockHeight updates the current block height.
	// This should be called when a new block is received.
	SetCurrentBlockHeight(height int64)
}

// ValidatorConfig contains configuration for the relay validator.
type ValidatorConfig struct {
	// AllowedSupplierAddresses is a list of supplier operator addresses
	// that this relayer is authorized to serve relays for.
	AllowedSupplierAddresses []string

	// GracePeriodExtraBlocks is additional grace period beyond on-chain config.
	GracePeriodExtraBlocks int64
}

// relayValidator implements RelayValidator.
type relayValidator struct {
	logger polylog.Logger
	config *ValidatorConfig

	// ringClient is used for ring signature verification.
	ringClient crypto.RingClient

	// sessionCache is used for session lookups.
	sessionCache cache.SessionCache

	// sharedParamCache is used for shared parameter lookups.
	sharedParamCache cache.SharedParamCache

	// currentBlockHeight is the latest known block height.
	currentBlockHeight int64
	blockHeightMu      sync.RWMutex

	// allowedSuppliers is a set of allowed supplier operator addresses.
	allowedSuppliers map[string]struct{}
}

// NewRelayValidator creates a new relay validator.
func NewRelayValidator(
	logger polylog.Logger,
	config *ValidatorConfig,
	ringClient crypto.RingClient,
	sessionCache cache.SessionCache,
	sharedParamCache cache.SharedParamCache,
) RelayValidator {
	allowedSuppliers := make(map[string]struct{})
	for _, addr := range config.AllowedSupplierAddresses {
		allowedSuppliers[addr] = struct{}{}
	}

	return &relayValidator{
		logger:           logging.ForComponent(logger, logging.ComponentRelayValidator),
		config:           config,
		ringClient:       ringClient,
		sessionCache:     sessionCache,
		sharedParamCache: sharedParamCache,
		allowedSuppliers: allowedSuppliers,
	}
}

// ValidateRelayRequest validates a relay request.
func (rv *relayValidator) ValidateRelayRequest(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
) error {
	// Basic validation
	if err := relayRequest.ValidateBasic(); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}

	meta := relayRequest.GetMeta()
	sessionHeader := meta.GetSessionHeader()

	// Check if the supplier is allowed
	supplierAddr := meta.GetSupplierOperatorAddress()
	if _, ok := rv.allowedSuppliers[supplierAddr]; !ok && len(rv.allowedSuppliers) > 0 {
		return fmt.Errorf("supplier %s is not allowed by this relayer", supplierAddr)
	}

	// Get target session block height (handles grace period)
	sessionBlockHeight, err := rv.getTargetSessionBlockHeight(ctx, relayRequest)
	if err != nil {
		return fmt.Errorf("session timing validation failed: %w", err)
	}

	// Verify ring signature
	if sigErr := rv.ringClient.VerifyRelayRequestSignature(ctx, relayRequest); sigErr != nil {
		return fmt.Errorf("ring signature verification failed: %w", sigErr)
	}

	// Verify session validity
	appAddress := sessionHeader.GetApplicationAddress()
	serviceID := sessionHeader.GetServiceId()

	session, err := rv.sessionCache.GetSession(ctx, appAddress, serviceID, sessionBlockHeight)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Verify session ID matches
	if session.SessionId != sessionHeader.GetSessionId() {
		return fmt.Errorf(
			"session ID mismatch, expected: %s, got: %s",
			session.SessionId,
			sessionHeader.GetSessionId(),
		)
	}

	// Verify supplier is in session
	supplierFound := false
	for _, supplier := range session.Suppliers {
		if supplier.OperatorAddress == supplierAddr {
			supplierFound = true
			break
		}
	}
	if !supplierFound {
		return fmt.Errorf("supplier %s not found in session", supplierAddr)
	}

	return nil
}

// CheckRewardEligibility checks if a relay is still eligible for rewards.
func (rv *relayValidator) CheckRewardEligibility(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
) error {
	currentHeight := rv.GetCurrentBlockHeight()
	if currentHeight == 0 {
		// If we don't have block height info, assume it's eligible
		return nil
	}

	sharedParams, err := rv.sharedParamCache.GetLatestSharedParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get shared params: %w", err)
	}

	sessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)

	// If current height >= claim window open height, relay is no longer eligible
	if currentHeight >= claimWindowOpenHeight {
		return fmt.Errorf(
			"session expired, must be before claim window open height (%d), current height is (%d)",
			claimWindowOpenHeight,
			currentHeight,
		)
	}

	return nil
}

// GetCurrentBlockHeight returns the current block height.
func (rv *relayValidator) GetCurrentBlockHeight() int64 {
	rv.blockHeightMu.RLock()
	defer rv.blockHeightMu.RUnlock()
	return rv.currentBlockHeight
}

// SetCurrentBlockHeight updates the current block height.
func (rv *relayValidator) SetCurrentBlockHeight(height int64) {
	rv.blockHeightMu.Lock()
	defer rv.blockHeightMu.Unlock()
	rv.currentBlockHeight = height
}

// getTargetSessionBlockHeight determines the block height to use for session lookup.
// It handles grace period logic.
func (rv *relayValidator) getTargetSessionBlockHeight(
	ctx context.Context,
	relayRequest *servicetypes.RelayRequest,
) (int64, error) {
	currentHeight := rv.GetCurrentBlockHeight()
	if currentHeight == 0 {
		// If we don't have block height info, use the session end height
		return relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight(), nil
	}

	sessionEndHeight := relayRequest.Meta.SessionHeader.GetSessionEndBlockHeight()

	// If session hasn't ended yet, use current height
	if sessionEndHeight >= currentHeight {
		return currentHeight, nil
	}

	// Session has ended, check grace period
	sharedParams, err := rv.sharedParamCache.GetLatestSharedParams(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get shared params: %w", err)
	}

	// Add extra grace period blocks if configured
	effectiveCurrentHeight := currentHeight
	if rv.config.GracePeriodExtraBlocks > 0 {
		// Subtract extra grace to be more lenient
		effectiveCurrentHeight = currentHeight - rv.config.GracePeriodExtraBlocks
		if effectiveCurrentHeight < sessionEndHeight {
			effectiveCurrentHeight = sessionEndHeight
		}
	}

	// Check if still within grace period
	if !sharedtypes.IsGracePeriodElapsed(sharedParams, sessionEndHeight, effectiveCurrentHeight) {
		// Within grace period, use session end height for lookup
		return sessionEndHeight, nil
	}

	return 0, fmt.Errorf(
		"session expired, session end height: %d, current height: %d (grace period elapsed)",
		sessionEndHeight,
		currentHeight,
	)
}
