package relayer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/pokt-network/poktroll/pkg/crypto"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// RelayProcessor handles the processing of relays including:
// - Deserializing relay requests
// - Building and signing relay responses
// - Calculating relay hashes
// - Checking mining difficulty
// - Publishing mined relays
type RelayProcessor interface {
	// ProcessRelay processes a served relay and returns a mined relay if applicable.
	// Returns nil if the relay doesn't meet mining difficulty.
	ProcessRelay(
		ctx context.Context,
		reqBody, respBody []byte,
		supplierAddr string,
		serviceID string,
		arrivalBlockHeight int64,
	) (*transport.MinedRelayMessage, error)

	// GetServiceDifficulty returns the current mining difficulty for a service.
	GetServiceDifficulty(ctx context.Context, serviceID string) ([]byte, error)

	// SetDifficultyProvider sets the difficulty provider for mining checks.
	SetDifficultyProvider(provider DifficultyProvider)
}

// DifficultyProvider provides mining difficulty targets for services.
type DifficultyProvider interface {
	// GetTargetHash returns the target hash for mining difficulty for a service.
	// Returns the base difficulty (all relays applicable) if service not found.
	GetTargetHash(ctx context.Context, serviceID string) ([]byte, error)
}

// ServiceComputeUnitsProvider provides compute units per relay for services.
type ServiceComputeUnitsProvider interface {
	// GetServiceComputeUnits returns the compute units per relay for a service.
	GetServiceComputeUnits(serviceID string) uint64
}

// RelaySignerKeyring provides relay signing capabilities.
type RelaySignerKeyring interface {
	// SignRelayResponse signs the relay response with the supplier's key.
	SignRelayResponse(
		ctx context.Context,
		response *servicetypes.RelayResponse,
		supplierOperatorAddr string,
	) ([]byte, error)
}

// relayProcessor implements RelayProcessor.
type relayProcessor struct {
	logger                      polylog.Logger
	publisher                   transport.MinedRelayPublisher
	signer                      RelaySignerKeyring
	difficultyProvider          DifficultyProvider
	serviceComputeUnitsProvider ServiceComputeUnitsProvider
	ringClient                  crypto.RingClient

	mu sync.RWMutex
}

// NewRelayProcessor creates a new relay processor.
func NewRelayProcessor(
	logger polylog.Logger,
	publisher transport.MinedRelayPublisher,
	signer RelaySignerKeyring,
	ringClient crypto.RingClient,
) *relayProcessor {
	return &relayProcessor{
		logger:     logging.ForComponent(logger, logging.ComponentRelayProcessor),
		publisher:  publisher,
		signer:     signer,
		ringClient: ringClient,
	}
}

// SetDifficultyProvider sets the difficulty provider.
func (rp *relayProcessor) SetDifficultyProvider(provider DifficultyProvider) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.difficultyProvider = provider
}

// SetServiceComputeUnitsProvider sets the service compute units provider.
func (rp *relayProcessor) SetServiceComputeUnitsProvider(provider ServiceComputeUnitsProvider) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.serviceComputeUnitsProvider = provider
}

// ProcessRelay processes a served relay.
func (rp *relayProcessor) ProcessRelay(
	ctx context.Context,
	reqBody, respBody []byte,
	supplierAddr string,
	serviceID string,
	arrivalBlockHeight int64,
) (*transport.MinedRelayMessage, error) {
	// Try to deserialize as a protobuf RelayRequest
	relayReq := &servicetypes.RelayRequest{}
	if err := relayReq.Unmarshal(reqBody); err != nil {
		// Not a valid relay request - this is common for non-relay traffic
		// that passes through the proxy. Just skip processing.
		rp.logger.Debug().
			Err(err).
			Str(logging.FieldServiceID, serviceID).
			Msg("request body is not a valid RelayRequest, skipping relay processing")
		return nil, nil
	}

	// Build relay response
	relayResp, err := rp.buildRelayResponse(ctx, relayReq, respBody, supplierAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to build relay response: %w", err)
	}

	// Create the full relay
	relay := &servicetypes.Relay{
		Req: relayReq,
		Res: relayResp,
	}

	// Calculate relay hash
	relayBz, err := relay.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal relay: %w", err)
	}

	relayHash := protocol.GetRelayHashFromBytes(relayBz)

	// Check mining difficulty
	isApplicable, err := rp.checkMiningDifficulty(ctx, serviceID, relayHash[:])
	if err != nil {
		rp.logger.Warn().
			Err(err).
			Str(logging.FieldServiceID, serviceID).
			Msg("failed to check mining difficulty, assuming applicable")
		isApplicable = true // Default to applicable on error
	}

	if !isApplicable {
		// Relay doesn't meet difficulty, skip publishing
		rp.logger.Debug().
			Str(logging.FieldServiceID, serviceID).
			Msg("relay does not meet mining difficulty, skipping")
		relaysSkippedDifficulty.WithLabelValues(serviceID).Inc()
		return nil, nil
	}

	// Extract session info from relay request
	sessionHeader := relayReq.Meta.SessionHeader
	sessionID := ""
	sessionStartHeight := int64(0)
	sessionEndHeight := int64(0)
	appAddress := ""

	if sessionHeader != nil {
		sessionID = sessionHeader.SessionId
		sessionStartHeight = sessionHeader.SessionStartBlockHeight
		sessionEndHeight = sessionHeader.SessionEndBlockHeight
		appAddress = sessionHeader.ApplicationAddress
	}

	// Build mined relay message
	msg := &transport.MinedRelayMessage{
		RelayHash:               relayHash[:],
		RelayBytes:              relayBz,
		ComputeUnitsPerRelay:    rp.getComputeUnits(serviceID),
		SessionId:               sessionID,
		SessionStartHeight:      sessionStartHeight,
		SessionEndHeight:        sessionEndHeight,
		SupplierOperatorAddress: supplierAddr,
		ServiceId:               serviceID,
		ApplicationAddress:      appAddress,
		ArrivalBlockHeight:      arrivalBlockHeight,
	}
	msg.SetPublishedAt()

	return msg, nil
}

// buildRelayResponse creates and signs a relay response.
func (rp *relayProcessor) buildRelayResponse(
	ctx context.Context,
	relayReq *servicetypes.RelayRequest,
	respBody []byte,
	supplierAddr string,
) (*servicetypes.RelayResponse, error) {
	// Create relay response with the backend response payload
	relayResp := &servicetypes.RelayResponse{
		Meta: servicetypes.RelayResponseMetadata{
			SessionHeader: relayReq.Meta.SessionHeader,
		},
		Payload: respBody,
	}

	// Calculate payload hash for signature efficiency (v0.1.25+)
	// This allows the response payload to be nil'd after signing
	// while still being verifiable via the payload hash
	payloadHash := sha256.Sum256(respBody)
	relayResp.PayloadHash = payloadHash[:]

	// Sign the response if signer is available
	if rp.signer != nil {
		sig, err := rp.signer.SignRelayResponse(ctx, relayResp, supplierAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to sign relay response: %w", err)
		}
		relayResp.Meta.SupplierOperatorSignature = sig
	}

	// Dehydrate the payload for storage efficiency (keep only hash + signature)
	// The signature was computed over the full payload, but we store nil
	// to reduce SMST and proof sizes
	relayResp.Payload = nil

	return relayResp, nil
}

// checkMiningDifficulty checks if a relay hash meets the mining difficulty.
func (rp *relayProcessor) checkMiningDifficulty(
	ctx context.Context,
	serviceID string,
	relayHash []byte,
) (bool, error) {
	targetHash, err := rp.GetServiceDifficulty(ctx, serviceID)
	if err != nil {
		return false, err
	}

	return protocol.IsRelayVolumeApplicable(relayHash, targetHash), nil
}

// GetServiceDifficulty returns the mining difficulty for a service.
func (rp *relayProcessor) GetServiceDifficulty(ctx context.Context, serviceID string) ([]byte, error) {
	rp.mu.RLock()
	provider := rp.difficultyProvider
	rp.mu.RUnlock()

	if provider == nil {
		// No provider, use base difficulty (all relays applicable)
		return protocol.BaseRelayDifficultyHashBz, nil
	}

	return provider.GetTargetHash(ctx, serviceID)
}

// getComputeUnits returns the compute units per relay for a service.
// Uses the configured provider, or falls back to 1 if not available.
func (rp *relayProcessor) getComputeUnits(serviceID string) uint64 {
	rp.mu.RLock()
	provider := rp.serviceComputeUnitsProvider
	rp.mu.RUnlock()

	if provider == nil {
		// No provider, fall back to 1 (will likely cause claim failures)
		return 1
	}

	return provider.GetServiceComputeUnits(serviceID)
}

// BaseDifficultyProvider always returns the base difficulty (all relays applicable).
// Useful for testing or when on-chain difficulty queries are not available.
type BaseDifficultyProvider struct{}

// GetTargetHash returns the base difficulty hash.
func (p *BaseDifficultyProvider) GetTargetHash(ctx context.Context, serviceID string) ([]byte, error) {
	return protocol.BaseRelayDifficultyHashBz, nil
}

// CachedDifficultyProvider caches difficulty targets per service.
type CachedDifficultyProvider struct {
	logger      polylog.Logger
	queryClient ServiceDifficultyQueryClient

	cache sync.Map // map[serviceID][]byte
}

// ServiceDifficultyQueryClient queries on-chain service difficulty.
type ServiceDifficultyQueryClient interface {
	// GetServiceRelayDifficulty returns the relay mining difficulty for a service.
	GetServiceRelayDifficulty(ctx context.Context, serviceID string) ([]byte, error)
}

// NewCachedDifficultyProvider creates a new cached difficulty provider.
func NewCachedDifficultyProvider(
	logger polylog.Logger,
	queryClient ServiceDifficultyQueryClient,
) *CachedDifficultyProvider {
	return &CachedDifficultyProvider{
		logger:      logging.ForComponent(logger, logging.ComponentDifficultyProvider),
		queryClient: queryClient,
	}
}

// GetTargetHash returns the cached difficulty target for a service.
func (p *CachedDifficultyProvider) GetTargetHash(ctx context.Context, serviceID string) ([]byte, error) {
	// Check cache first
	if cached, ok := p.cache.Load(serviceID); ok {
		return cached.([]byte), nil
	}

	// Query on-chain
	if p.queryClient == nil {
		return protocol.BaseRelayDifficultyHashBz, nil
	}

	target, err := p.queryClient.GetServiceRelayDifficulty(ctx, serviceID)
	if err != nil {
		p.logger.Warn().
			Err(err).
			Str(logging.FieldServiceID, serviceID).
			Msg("failed to query service difficulty, using base")
		return protocol.BaseRelayDifficultyHashBz, nil
	}

	// Cache the result
	p.cache.Store(serviceID, target)

	return target, nil
}

// InvalidateCache clears the difficulty cache for a service.
func (p *CachedDifficultyProvider) InvalidateCache(serviceID string) {
	p.cache.Delete(serviceID)
}

// InvalidateAllCache clears all cached difficulties.
func (p *CachedDifficultyProvider) InvalidateAllCache() {
	p.cache.Range(func(key, value interface{}) bool {
		p.cache.Delete(key)
		return true
	})
}

// =============================================================================
// Cached Service Compute Units Provider
// =============================================================================

// ServiceQueryClient queries on-chain service data.
type ServiceQueryClient interface {
	// GetService returns the service entity for a service ID.
	GetService(ctx context.Context, serviceID string) (sharedtypes.Service, error)
}

// CachedServiceComputeUnitsProvider caches compute units per service from on-chain data.
type CachedServiceComputeUnitsProvider struct {
	logger      polylog.Logger
	queryClient ServiceQueryClient

	cache sync.Map // map[serviceID]uint64
}

// NewCachedServiceComputeUnitsProvider creates a new cached compute units provider.
func NewCachedServiceComputeUnitsProvider(
	logger polylog.Logger,
	queryClient ServiceQueryClient,
) *CachedServiceComputeUnitsProvider {
	return &CachedServiceComputeUnitsProvider{
		logger:      logging.ForComponent(logger, logging.ComponentRelayProcessor),
		queryClient: queryClient,
	}
}

// GetServiceComputeUnits returns the compute units per relay for a service.
// If the service is not cached, it queries the chain synchronously.
func (p *CachedServiceComputeUnitsProvider) GetServiceComputeUnits(serviceID string) uint64 {
	// Check cache first
	if cached, ok := p.cache.Load(serviceID); ok {
		return cached.(uint64)
	}

	// Query on-chain (synchronously - consider background refresh for production)
	if p.queryClient == nil {
		p.logger.Warn().
			Str(logging.FieldServiceID, serviceID).
			Msg("no query client available, using default compute units")
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	service, err := p.queryClient.GetService(ctx, serviceID)
	if err != nil {
		p.logger.Warn().
			Err(err).
			Str(logging.FieldServiceID, serviceID).
			Msg("failed to query service compute units, using default")
		return 1
	}

	computeUnits := service.ComputeUnitsPerRelay
	p.cache.Store(serviceID, computeUnits)

	p.logger.Debug().
		Str(logging.FieldServiceID, serviceID).
		Uint64("compute_units", computeUnits).
		Msg("cached service compute units from chain")

	return computeUnits
}

// PreloadServiceComputeUnits preloads compute units for a list of services.
// Call this at startup to avoid synchronous queries during relay processing.
func (p *CachedServiceComputeUnitsProvider) PreloadServiceComputeUnits(ctx context.Context, serviceIDs []string) {
	for _, serviceID := range serviceIDs {
		if p.queryClient == nil {
			continue
		}

		service, err := p.queryClient.GetService(ctx, serviceID)
		if err != nil {
			p.logger.Warn().
				Err(err).
				Str(logging.FieldServiceID, serviceID).
				Msg("failed to preload service compute units")
			continue
		}

		p.cache.Store(serviceID, service.ComputeUnitsPerRelay)
		p.logger.Info().
			Str(logging.FieldServiceID, serviceID).
			Uint64("compute_units", service.ComputeUnitsPerRelay).
			Msg("preloaded service compute units")
	}
}

// InvalidateCache clears the compute units cache for a service.
func (p *CachedServiceComputeUnitsProvider) InvalidateCache(serviceID string) {
	p.cache.Delete(serviceID)
}

// InvalidateAllCache clears all cached compute units.
func (p *CachedServiceComputeUnitsProvider) InvalidateAllCache() {
	p.cache.Range(func(key, value interface{}) bool {
		p.cache.Delete(key)
		return true
	})
}

// Verify interface compliance.
var _ RelayProcessor = (*relayProcessor)(nil)
var _ DifficultyProvider = (*BaseDifficultyProvider)(nil)
var _ DifficultyProvider = (*CachedDifficultyProvider)(nil)
var _ ServiceComputeUnitsProvider = (*CachedServiceComputeUnitsProvider)(nil)
