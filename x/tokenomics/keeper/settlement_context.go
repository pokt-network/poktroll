package keeper

import (
	"context"
	"fmt"
	"math/big"
	"slices"

	cosmoslog "cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/pkg/encoding"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// claimSettlementContext tracks the state and metadata for a claim during settlement.
//
// It includes but is not limited to:
// - Holding the result of settlement operations for a single claim: mint, burn, and transfer.
// - Indicating if the claim was settled or expired without settlement.
// - Recording the total relays and compute units associated with the claim.
type claimSettlementContext struct {
	// settlementResult holds the tokenomics module operations (mint, burn, transfer)
	// to be executed when settling this claim.
	settlementResult *tokenomicstypes.ClaimSettlementResult

	// If true, the claim was successfully settled;
	// Iff false, the claim expired without settlement.
	isSettled bool

	// numClaimRelays is the total number of claimed relays by the supplier.
	numClaimRelays uint64

	// numClaimComputeUnits is the total number of claimed compute units by the supplier.
	numClaimComputeUnits uint64

	// actualSettlementCoin is the settlement amount after overservicing cap,
	// before mint_ratio is applied. Used to populate EventClaimSettled.SettledUpokt.
	actualSettlementCoin cosmostypes.Coin
}

// claimTelemetryData holds telemetry data for a single claim to be emitted after settlement processing.
// This struct is used to batch telemetry emission instead of using deferred calls in a loop,
// which avoids excessive stack frame allocation with high claim volumes.
type claimTelemetryData struct {
	stage           prooftypes.ClaimProofStage
	serviceId       string
	appAddr         string
	supplierAddr    string
	numRelays       uint64
	numComputeUnits uint64
	err             error
}

// appSessionKey is a composite key for counting suppliers per (application, session) pair.
// Uses a struct key instead of fmt.Sprintf to avoid heap allocations.
type appSessionKey struct {
	appAddress string
	sessionId  string
}

// sessionBudget holds the precomputed budget allocation for a single (application,
// session) group, used to demote the per-supplier head-split from a hard ceiling to a
// guaranteed floor (settlement budget redistribution).
//
// All amounts are in "stake terms" — the settlement amount plus its per-claim global
// inflation reimbursement — matching the units ensureClaimAmountLimits compares against
// the application's stake. Populated in Phase 1.5 of SettlePendingClaims, consumed in
// Phase 2.
type sessionBudget struct {
	// numSuppliers is the actual number of suppliers that submitted a claim for this
	// (app, session) pair (== the value GetActualSupplierCount returns).
	numSuppliers int64
	// floor is the guaranteed per-supplier minimum settlement (B / numSuppliers), where
	// B is the application's per-session budget (appStake / numPendingSessions, optionally
	// clamped by per_session_spend_limit). Serving at or below the floor is always paid in
	// full.
	floor math.Int
	// unused is the budget left on the table by suppliers that claimed below their floor:
	// Σ max(0, floor − claimStake_i). This is what gets redistributed to overservicers.
	unused math.Int
	// totalExcess is the total demand above the floor: Σ max(0, claimStake_i − floor).
	// Used as the denominator when apportioning `unused` in proportion to each
	// overservicer's excess.
	totalExcess math.Int
	// spendLimitExceeded records whether the application's per_session_spend_limit (rather
	// than its stake) bound B, so Phase 2 can set the flag on EventApplicationOverserviced.
	spendLimitExceeded bool
}

// settlementContext maintains a cache of all entities involved in the claim settlement process.
// This structure optimizes claim processing performance by eliminating redundant KV store operations.
type settlementContext struct {
	logger cosmoslog.Logger
	keeper *Keeper

	// Maintain both a map and a slice for suppliers to provide:
	// 1. Fast O(1) lookup by address via the map for claim processing efficiency
	// 2. Consistent ordering via the slice to ensure deterministic processing results
	// 3. Preservation of the original insertion order for reporting and state updates
	// This dual structure prevents repeated KV store lookups while maintaining processing order
	supplierMap      map[string]int
	settledSuppliers []*sharedtypes.Supplier

	// Maintain both a map and a slice for applications to provide:
	// 1. Fast O(1) lookup by address via the map during claim processing
	// 2. Consistent ordering via the slice to ensure deterministic settlement results
	// 3. Preservation of the original processing order for state updates
	// This prevents repeated application store lookups while keeping processing consistent
	applicationMap             map[string]int
	settledApplications        []*apptypes.Application
	applicationInitialStakeMap map[string]cosmostypes.Coin

	// Cache of services and relay mining difficulties encountered during settlement.
	// Each service is only fetched once from the store and reused for all related claims.
	// The service ID is used as the key for the maps
	serviceMap               map[string]*sharedtypes.Service
	relayMiningDifficultyMap map[string]servicetypes.RelayMiningDifficulty

	// Cache of the compute_units_per_relay that was effective at each session-start
	// height, keyed by "serviceId@sessionStartHeight" (same composite key as
	// relayMiningDifficultyMap). Pinning cupr to session-start — rather than reading the
	// live service cupr — prevents a mid-session cupr change from discarding in-flight
	// claims at settlement (the settlement-side counterpart to the session-start pin
	// applied at claim creation).
	computeUnitsPerRelayMap map[string]uint64

	// Cache of parameters used during the settlement process to prevent repeated KV store lookups.
	sharedParams     sharedtypes.Params
	tokenomicsParams tokenomicstypes.Params

	// globalInflationPerClaimRat memoizes tokenomicsParams.GlobalInflationPerClaim as a
	// big.Rat. The conversion (strconv.FormatFloat + big.Rat.SetString) is identical for
	// every claim in a block, but was previously redone up to 4x per claim (once in Phase
	// 1.5, plus once in claimStakeTermsAmount and twice in supplierAppStakeToMaxSettlementAmount
	// during Phase 2) — tens of thousands of redundant conversions per mainnet settlement
	// block. Populated lazily by getGlobalInflationPerClaimRat.
	//
	// READ-ONLY once set: it is handed to callers by pointer, and every consumer builds
	// results with new(big.Rat), so it is never mutated in place.
	globalInflationPerClaimRat *big.Rat

	// validatorRewardAccumulator collects per-TLM proposer amounts across all claims
	// during a settlement batch. Keyed by SettlementOpReason. After all claims are
	// processed, these are flushed via a single distributeValidatorRewards call per key,
	// eliminating per-claim precision loss (#1758).
	validatorRewardAccumulator map[tokenomicstypes.SettlementOpReason]math.Int

	// supplierCountPerAppSession tracks how many suppliers submitted claims for each
	// (application, session) pair. Populated during the claim collection phase and
	// consumed during settlement to replace the NumSuppliersPerSession governance param
	// with the actual number of claimants, ensuring fair budget distribution.
	supplierCountPerAppSession map[appSessionKey]int64

	// budgetPerAppSession holds the precomputed per-(application, session) budget
	// allocation (floor + redistributable unused budget). Populated in Phase 1.5 of
	// SettlePendingClaims and consumed in Phase 2 by ensureClaimAmountLimits. Lookup-only
	// from the consensus path — NEVER range over this map to produce output or state
	// (map iteration order is non-deterministic).
	budgetPerAppSession map[appSessionKey]*sessionBudget
}

// NewSettlementContext creates a new settlement context with all necessary caches initialized.
// This function establishes an efficient memory structure that prevents redundant KV store operations
// during claim processing, improving settlement performance for high volume operations.
func NewSettlementContext(
	ctx context.Context,
	tokenomicsKeeper *Keeper,
	logger cosmoslog.Logger,
) *settlementContext {
	// Pre-allocate maps and slices with reasonable capacity estimates based on current network size.
	// This avoids rehashing and reallocation during claim processing with high claim volumes.
	// Current network: ~5,250 suppliers, ~300 services, ~200 applications.
	const (
		estimatedSuppliers    = 5500
		estimatedApplications = 250
		estimatedServices     = 350
	)

	return &settlementContext{
		keeper: tokenomicsKeeper,
		logger: logger.With("module", "settlement_context"),

		supplierMap:      make(map[string]int, estimatedSuppliers),
		settledSuppliers: make([]*sharedtypes.Supplier, 0, estimatedSuppliers),

		applicationMap:             make(map[string]int, estimatedApplications),
		settledApplications:        make([]*apptypes.Application, 0, estimatedApplications),
		applicationInitialStakeMap: make(map[string]cosmostypes.Coin, estimatedApplications),

		serviceMap:               make(map[string]*sharedtypes.Service, estimatedServices),
		relayMiningDifficultyMap: make(map[string]servicetypes.RelayMiningDifficulty, estimatedServices),
		computeUnitsPerRelayMap:  make(map[string]uint64, estimatedServices),

		sharedParams:     tokenomicsKeeper.sharedKeeper.GetParams(ctx),
		tokenomicsParams: tokenomicsKeeper.GetParams(ctx),

		// Initialize the validator reward accumulator with capacity for the 2 known TLM op reasons
		// (TLM_RELAY_BURN_EQUALS_MINT and TLM_GLOBAL_MINT).
		validatorRewardAccumulator: make(map[tokenomicstypes.SettlementOpReason]math.Int, 2),

		// Initialize the supplier count map. Estimated capacity: each app typically has
		// 1 session being settled per block, but may have more with multiple services.
		supplierCountPerAppSession: make(map[appSessionKey]int64, estimatedApplications),

		// Initialize the per-(app, session) budget map with the same capacity estimate:
		// one settling session per application per settlement block.
		budgetPerAppSession: make(map[appSessionKey]*sessionBudget, estimatedApplications),
	}
}

// FlushAllActorsToStore batch save all accumulated applications and suppliers to the store.
// It is intended to be called AFTER all claims have been processed.

// This optimization:
//   - Reduces redundant writes to state storage by updating each record only once.
//   - Avoids repeated KV store operations for applications and suppliers involved in multiple claims
func (sctx *settlementContext) FlushAllActorsToStore(ctx context.Context) {
	logger := sctx.logger.With("method", "FlushAllActorsToStore")

	// Flush all Application records to the store
	for _, application := range sctx.settledApplications {
		sctx.keeper.applicationKeeper.SetApplication(ctx, *application)
		logger.Info(fmt.Sprintf("updated onchain application record with address %q", application.Address))
	}
	logger.Info(fmt.Sprintf("updated %d onchain application records", len(sctx.settledApplications)))

	// Flush all Supplier records to the store
	for _, supplier := range sctx.settledSuppliers {
		sctx.keeper.supplierKeeper.SetDehydratedSupplier(ctx, *supplier)
		logger.Info(fmt.Sprintf("updated onchain supplier record with address %q", supplier.OperatorAddress))
	}
	logger.Info(fmt.Sprintf("updated %d onchain supplier records", len(sctx.settledSuppliers)))
}

// ClaimCacheWarmUp warms up the settlement context's cache by based on the claim's properties.
func (sctx *settlementContext) ClaimCacheWarmUp(ctx context.Context, claim *prooftypes.Claim) error {
	if claim.SessionHeader == nil {
		return tokenomicstypes.ErrTokenomicsClaimSessionHeaderNil
	}

	// Cache service and difficulty using the session start height to get historical difficulty
	serviceId := claim.SessionHeader.ServiceId
	sessionStartHeight := claim.SessionHeader.SessionStartBlockHeight
	if err := sctx.cacheServiceAndDifficulty(ctx, serviceId, sessionStartHeight); err != nil {
		return err
	}

	// Cache application
	applicationAddress := claim.SessionHeader.ApplicationAddress
	if err := sctx.cacheApplication(ctx, applicationAddress); err != nil {
		return err
	}

	// Cache supplier
	supplierOperatorAddress := claim.SupplierOperatorAddress
	if err := sctx.cacheSupplier(ctx, supplierOperatorAddress, serviceId); err != nil {
		return err
	}

	return nil
}

// GetApplicationInitialStake retrieves the initial stake of an application at the
// beginning of the settlement process.
func (sctx *settlementContext) GetApplicationInitialStake(appAddress string) (cosmostypes.Coin, error) {
	if stake, ok := sctx.applicationInitialStakeMap[appAddress]; ok {
		return stake, nil
	}

	sctx.logger.Error(fmt.Sprintf("initial stake for application with address %q not found", appAddress))
	return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsApplicationNotFound.Wrapf(
		"initial stake for application with address %q not found", appAddress,
	)
}

// GetRelayMiningDifficulty retrieves the cached relay mining difficulty for a specific service at a given session height.
func (sctx *settlementContext) GetRelayMiningDifficulty(serviceId string, sessionStartHeight int64) (servicetypes.RelayMiningDifficulty, error) {
	// Generate cache key that includes session height
	cacheKey := fmt.Sprintf("%s@%d", serviceId, sessionStartHeight)

	if relayMiningDifficulty, ok := sctx.relayMiningDifficultyMap[cacheKey]; ok {
		return relayMiningDifficulty, nil
	}

	sctx.logger.Error(fmt.Sprintf("relay mining difficulty for service with ID %q at session start height %d not found", serviceId, sessionStartHeight))
	return servicetypes.RelayMiningDifficulty{}, tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf(
		"relay mining difficulty for service with ID %q at session start height %d not found", serviceId, sessionStartHeight,
	)
}

// GetServiceComputeUnitsPerRelay retrieves the cached compute_units_per_relay that was
// effective at the given session start height for a specific service. This is the
// session-start-pinned cupr used to validate claims at settlement, mirroring
// GetRelayMiningDifficulty.
func (sctx *settlementContext) GetServiceComputeUnitsPerRelay(serviceId string, sessionStartHeight int64) (uint64, error) {
	// Generate cache key that includes session height
	cacheKey := fmt.Sprintf("%s@%d", serviceId, sessionStartHeight)

	if computeUnitsPerRelay, ok := sctx.computeUnitsPerRelayMap[cacheKey]; ok {
		return computeUnitsPerRelay, nil
	}

	sctx.logger.Error(fmt.Sprintf("compute units per relay for service with ID %q at session start height %d not found", serviceId, sessionStartHeight))
	return 0, tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf(
		"compute units per relay for service with ID %q at session start height %d not found", serviceId, sessionStartHeight,
	)
}

// getGlobalInflationPerClaimRat returns the settlement block's global_inflation_per_claim
// as a big.Rat, performing the float64->Rat conversion at most once per settlement block.
//
// The returned pointer is shared by every caller and MUST NOT be mutated.
func (sctx *settlementContext) getGlobalInflationPerClaimRat() (*big.Rat, error) {
	if sctx.globalInflationPerClaimRat != nil {
		return sctx.globalInflationPerClaimRat, nil
	}

	globalInflationPerClaimRat, err := encoding.Float64ToRat(sctx.tokenomicsParams.GlobalInflationPerClaim)
	if err != nil {
		return nil, err
	}
	sctx.globalInflationPerClaimRat = globalInflationPerClaimRat

	return globalInflationPerClaimRat, nil
}

// GetService retrieves a cached service by its ID.
func (sctx *settlementContext) GetService(serviceId string) (*sharedtypes.Service, error) {
	if service, ok := sctx.serviceMap[serviceId]; ok {
		return service, nil
	}

	sctx.logger.Error(fmt.Sprintf("service with ID %q not found", serviceId))
	return nil, tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf(
		"service with ID %q not found", serviceId,
	)
}

// GetSharedParams returns the cached shared parameters used during the settlement process.
func (sctx *settlementContext) GetSharedParams() sharedtypes.Params {
	return sctx.sharedParams
}

// GetTokenomicsParams returns the cached tokenomics parameters used during the settlement process.
func (sctx *settlementContext) GetTokenomicsParams() tokenomicstypes.Params {
	return sctx.tokenomicsParams
}

// GetValidatorRewardAccumulator returns the shared accumulator map used by TLMs
// to batch validator reward amounts across claims (#1758).
func (sctx *settlementContext) GetValidatorRewardAccumulator() map[tokenomicstypes.SettlementOpReason]math.Int {
	return sctx.validatorRewardAccumulator
}

// GetApplication retrieves a cached application by its address.
func (sctx *settlementContext) GetApplication(appAddress string) (*apptypes.Application, error) {
	if idx, ok := sctx.applicationMap[appAddress]; ok {
		return sctx.settledApplications[idx], nil
	}

	sctx.logger.Error(fmt.Sprintf("application with address %q not found", appAddress))
	return nil, tokenomicstypes.ErrTokenomicsApplicationNotFound.Wrapf(
		"application with address %q not found", appAddress)
}

// GetSupplier retrieves a cached supplier by its operator address.
func (sctx *settlementContext) GetSupplier(supplierOperatorAddress string) (*sharedtypes.Supplier, error) {
	if idx, ok := sctx.supplierMap[supplierOperatorAddress]; ok {
		return sctx.settledSuppliers[idx], nil
	}

	sctx.logger.Error(fmt.Sprintf("supplier with address %q not found", supplierOperatorAddress))
	return nil, tokenomicstypes.ErrTokenomicsSupplierNotFound.Wrapf(
		"supplier with address %q not found", supplierOperatorAddress,
	)
}

// GetSettledApplications returns all applications that have been modified during the settlement process.
//
// These applications will returned in the order they were added to the settlement context.
func (sctx *settlementContext) GetSettledApplications() []*apptypes.Application {
	return sctx.settledApplications
}

// GetSettledSuppliers returns all suppliers that have been modified during the settlement process.
//
// These suppliers will returned in the order they were added to the settlement context.
func (sctx *settlementContext) GetSettledSuppliers() []*sharedtypes.Supplier {
	return sctx.settledSuppliers
}

// cacheSupplier ensures the supplier for a claim is cached in the settlement context.
//
// This prevents repeated KV store lookups for the same supplier across multiple claims.
func (sctx *settlementContext) cacheSupplier(
	ctx context.Context,
	supplierOperatorAddress string,
	serviceId string,
) error {
	// Supplier service configuration handling:
	// - Suppliers are hydrated with service configurations on demand (lazily)
	// - Each new service ID in a claim is added to the supplier's configuration only once
	// - This prevents unnecessary duplicate service configurations in memory
	// - In the worst case, a supplier will have all its service configurations loaded
	//   if claims are processed for all services the supplier supports
	if idx, ok := sctx.supplierMap[supplierOperatorAddress]; ok {
		cachedSupplier := sctx.settledSuppliers[idx]

		// Supplier is cached, ensure that it has a service configuration corresponding
		// to the claim's service ID.
		sctx.cacheSupplierServiceConfig(ctx, cachedSupplier, serviceId)

		// Supplier is already cached
		return nil
	}

	// Retrieve the onchain staked dehydrated supplier record:
	// - We only fetch the dehydrated supplier (without all service configs)
	// - This fetch happens only once - when we process the first claim from this supplier
	// - Additional service configs will be hydrated on demand as needed
	// - This optimizes memory usage and reduces redundant store lookups
	supplier, isSupplierFound := sctx.keeper.supplierKeeper.GetDehydratedSupplier(ctx, supplierOperatorAddress)
	if !isSupplierFound {
		sctx.logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierOperatorAddress))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	// Hydrate the supplier service configuration with the claim's service ID.
	// This is needed to ensure the dehydrated supplier has the correct service
	// revenue share configuration for the claim settlement.
	supplier.Services = sctx.keeper.supplierKeeper.GetSupplierActiveServiceConfig(ctx, &supplier, serviceId)

	// Store supplier in cache for future claim processing
	idx := len(sctx.settledSuppliers)
	sctx.supplierMap[supplierOperatorAddress] = idx
	sctx.settledSuppliers = append(sctx.settledSuppliers, &supplier)

	return nil
}

// cacheSupplierServiceConfig ensures the supplier service configuration for a claim
// is cached in the settlement context.
func (sctx *settlementContext) cacheSupplierServiceConfig(
	ctx context.Context,
	supplier *sharedtypes.Supplier,
	serviceId string,
) {
	serviceConfigIdx := slices.IndexFunc(supplier.Services, func(s *sharedtypes.SupplierServiceConfig) bool {
		return s.ServiceId == serviceId
	})

	// Service configuration already cached, no need to update
	if serviceConfigIdx >= 0 {
		return
	}

	// Hydrate the supplier service configuration with the claim's service ID.
	// This is needed to ensure the dehydrated supplier has the correct service
	// revenue share configuration for the claim settlement.
	supplier.Services = append(
		supplier.Services,
		sctx.keeper.supplierKeeper.GetSupplierActiveServiceConfig(ctx, supplier, serviceId)...,
	)
}

// cacheApplication ensures the application for a claim is cached in the settlement context.
//
// This prevents repeated KV store lookups for the same application across multiple claims.
func (sctx *settlementContext) cacheApplication(ctx context.Context, appAddress string) error {
	if _, ok := sctx.applicationMap[appAddress]; ok {
		// Application is already cached
		return nil
	}

	// Retrieve the onchain staked application record
	application, isAppFound := sctx.keeper.applicationKeeper.GetApplication(ctx, appAddress)
	if !isAppFound {
		sctx.logger.Warn(fmt.Sprintf("application for claim with address %q not found", appAddress))
		return tokenomicstypes.ErrTokenomicsApplicationNotFound
	}

	// Store application in cache for future claim processing
	idx := len(sctx.settledApplications)
	sctx.applicationMap[appAddress] = idx
	sctx.settledApplications = append(sctx.settledApplications, &application)

	// Cache the initial stake for the application at the beginning of settlement to:
	// - Ensure the claim amount limits are not exceeded by the Suppliers.
	// - Use a consistent reference value for all claims involving this application.
	// - Prevent accounting issues that would occur if we used the continuously changing stake value
	//   during the settlement process.
	sctx.applicationInitialStakeMap[appAddress] = *application.Stake

	return nil
}

// IncrementSupplierCount increments the supplier count for the given (app, session) pair.
// Called during the claim collection phase to track actual claimants.
func (sctx *settlementContext) IncrementSupplierCount(appAddress, sessionId string) {
	key := appSessionKey{appAddress: appAddress, sessionId: sessionId}
	sctx.supplierCountPerAppSession[key]++
}

// GetActualSupplierCount returns the number of suppliers that submitted claims
// for the given (app, session) pair. Falls back to 1 if no count is found
// (should never happen since counts are populated during collection phase).
func (sctx *settlementContext) GetActualSupplierCount(appAddress, sessionId string) int64 {
	key := appSessionKey{appAddress: appAddress, sessionId: sessionId}
	count, ok := sctx.supplierCountPerAppSession[key]
	if !ok || count == 0 {
		sctx.logger.Warn(fmt.Sprintf(
			"no supplier count found for app %q session %q, defaulting to 1",
			appAddress, sessionId,
		))
		return 1
	}
	return count
}

// AccumulateClaimBudget performs the Phase 1.5 budget accounting for a single claim: it
// derives the claim's stake-terms amount (settlement amount + per-claim global inflation)
// and folds it into its (application, session) group's floor/unused/excess totals.
//
// The claim's caches (service, difficulty, application) MUST already be warm — call
// ClaimCacheWarmUp first. Returns an error if the claim cannot be priced or the group's
// budget cannot be initialized; the caller treats such claims exactly as before (left for
// Phase 2 to discard) so they contribute nothing to the budget math.
func (sctx *settlementContext) AccumulateClaimBudget(ctx context.Context, claim *prooftypes.Claim) error {
	sessionHeader := claim.GetSessionHeader()
	if sessionHeader == nil {
		return tokenomicstypes.ErrTokenomicsClaimSessionHeaderNil
	}

	relayMiningDifficulty, err := sctx.GetRelayMiningDifficulty(
		sessionHeader.GetServiceId(),
		sessionHeader.GetSessionStartBlockHeight(),
	)
	if err != nil {
		return err
	}

	claimSettlementCoin, err := claim.GetClaimeduPOKT(sctx.sharedParams, relayMiningDifficulty)
	if err != nil {
		return err
	}

	globalInflationPerClaimRat, err := sctx.getGlobalInflationPerClaimRat()
	if err != nil {
		return err
	}
	claimStakeAmt := claimStakeTermsAmount(claimSettlementCoin, globalInflationPerClaimRat)

	sb, err := sctx.getOrInitSessionBudget(
		ctx,
		sessionHeader.GetApplicationAddress(),
		sessionHeader.GetSessionId(),
		sessionHeader.GetSessionEndBlockHeight(),
	)
	if err != nil {
		return err
	}

	// Commutative integer accumulation — order-independent, hence consensus-safe.
	if claimStakeAmt.LT(sb.floor) {
		sb.unused = sb.unused.Add(sb.floor.Sub(claimStakeAmt))
	} else {
		sb.totalExcess = sb.totalExcess.Add(claimStakeAmt.Sub(sb.floor))
	}

	return nil
}

// getOrInitSessionBudget lazily computes and caches the budget allocation for an
// (application, session) group. The floor (B / N) is a pure function of the application's
// initial stake, the per-session concurrency (numPendingSessions at the claim's session-end
// params epoch), the actual number of claiming suppliers, and the application's optional
// per-session spend limit — none of which depend on the individual claim — so it is
// computed once per group and reused.
//
// This mirrors the legacy floor computation in ensureClaimAmountLimits exactly, so that
// overservicing_bonus_multiplier == 1 reproduces the pre-change settlement byte-for-byte.
func (sctx *settlementContext) getOrInitSessionBudget(
	ctx context.Context,
	appAddress, sessionId string,
	sessionEndBlockHeight int64,
) (*sessionBudget, error) {
	key := appSessionKey{appAddress: appAddress, sessionId: sessionId}
	if sb, ok := sctx.budgetPerAppSession[key]; ok {
		return sb, nil
	}

	appStake, err := sctx.GetApplicationInitialStake(appAddress)
	if err != nil {
		return nil, err
	}
	application, err := sctx.GetApplication(appAddress)
	if err != nil {
		return nil, err
	}

	numSuppliers := sctx.GetActualSupplierCount(appAddress, sessionId)

	// numPendingSessions is read at the params epoch effective for the claim's own session
	// end height (not live): a num_blocks_per_session change between a session and its
	// settlement otherwise re-divides the app stake by a different concurrency (#543, F2).
	budgetParams := sctx.keeper.sharedKeeper.GetParamsAtHeight(ctx, sessionEndBlockHeight)
	numPendingSessions := sharedtypes.GetNumPendingSessions(&budgetParams)

	// floor = appStake / numPendingSessions / numSuppliers.
	// Divide sessions first, then suppliers — matches the legacy integer-truncation order.
	floor := appStake.Amount.
		Quo(math.NewInt(numPendingSessions)).
		Quo(math.NewInt(numSuppliers))

	// Apply the application's per-session spend limit, if set: it caps the per-session
	// budget B (= appStake/numPendingSessions), which is then divided among suppliers.
	spendLimitExceeded := false
	if application.PerSessionSpendLimit != nil &&
		application.PerSessionSpendLimit.Amount.GT(math.ZeroInt()) {
		if application.PerSessionSpendLimit.Denom != pocket.DenomuPOKT {
			return nil, tokenomicstypes.ErrTokenomicsApplicationNewStakeInvalid.Wrapf(
				"application %s has per_session_spend_limit with invalid denom %q (expected %q)",
				appAddress, application.PerSessionSpendLimit.Denom, pocket.DenomuPOKT,
			)
		}
		perSessionBudget := application.PerSessionSpendLimit.Amount
		stakePerSession := appStake.Amount.Quo(math.NewInt(numPendingSessions))
		if perSessionBudget.GT(stakePerSession) {
			perSessionBudget = stakePerSession
		}
		spendLimitFloor := perSessionBudget.Quo(math.NewInt(numSuppliers))
		if spendLimitFloor.LT(floor) {
			floor = spendLimitFloor
			spendLimitExceeded = true
		}
	}

	sb := &sessionBudget{
		numSuppliers:       numSuppliers,
		floor:              floor,
		unused:             math.ZeroInt(),
		totalExcess:        math.ZeroInt(),
		spendLimitExceeded: spendLimitExceeded,
	}
	sctx.budgetPerAppSession[key] = sb
	return sb, nil
}

// cacheServiceAndDifficulty ensures the service and its relay mining difficulty are cached in the settlement context.
//
// This prevents repeated KV store lookups across multiple claims targeting the same service at the same session height.
func (sctx *settlementContext) cacheServiceAndDifficulty(ctx context.Context, serviceId string, sessionStartHeight int64) error {
	// Generate cache key that includes session height
	cacheKey := fmt.Sprintf("%s@%d", serviceId, sessionStartHeight)

	if _, ok := sctx.relayMiningDifficultyMap[cacheKey]; ok {
		// Difficulty for this service/session combination is already cached.
		// The difficulty and compute-units-per-relay maps are written together at the
		// very end of this function, after every fallible step, so difficulty presence
		// implies cupr presence — GetServiceComputeUnitsPerRelay relies on this.
		return nil
	}

	// Retrieve the service record (service doesn't change per session, so cache by serviceId only)
	if _, ok := sctx.serviceMap[serviceId]; !ok {
		service, isServiceFound := sctx.keeper.serviceKeeper.GetService(ctx, serviceId)
		if !isServiceFound {
			sctx.logger.Warn(fmt.Sprintf("service with ID %q not found", serviceId))
			return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", serviceId)
		}
		sctx.serviceMap[serviceId] = &service
	}

	// Retrieve difficulty that was effective at the session start height
	relayMiningDifficulty, found := sctx.keeper.serviceKeeper.GetRelayMiningDifficultyAtHeight(ctx, serviceId, sessionStartHeight)
	if !found {
		targetNumRelays := sctx.keeper.serviceKeeper.GetParams(ctx).TargetNumRelays
		relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(
			ctx,
			sctx.logger,
			serviceId,
			targetNumRelays,
			targetNumRelays,
		)
	}
	// Retrieve the compute_units_per_relay that was effective at the session start
	// height. This mirrors the difficulty lookup above so that settlement validates
	// claims against the cupr that was live when the session started — not the current
	// (possibly changed) service cupr. `found` is false only if the service does not
	// exist, which is already guarded above; the explicit check keeps that invariant
	// local so a future refactor cannot silently cache cupr=0 (which would fail the
	// numRelays*cupr equality check and discard the claim — the exact forfeit this
	// session-start pin exists to prevent).
	//
	// DEV_NOTE: resolved BEFORE the difficulty is cached so that the two maps are only
	// ever written together. The early return at the top of this function keys off the
	// difficulty map alone, so writing difficulty first and then failing here would leave
	// the caches inconsistent AND make the next (idempotent) warm-up return nil with cupr
	// still missing — surfacing later as an opaque settlement error.
	computeUnitsPerRelay, found := sctx.keeper.serviceKeeper.GetServiceComputeUnitsPerRelayAtHeight(ctx, serviceId, sessionStartHeight)
	if !found {
		sctx.logger.Warn(fmt.Sprintf("compute units per relay for service with ID %q not found", serviceId))
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("compute units per relay for service with ID %q not found", serviceId)
	}

	// A cupr of 0 can never be legitimate: MsgAddService validation requires >= 1, so a
	// zero here means a corrupt or orphaned history entry (MustUnmarshal on a nil/!ok
	// value yields a zero-valued struct rather than panicking with gogoproto). Caching it
	// would fail the numRelays*cupr equality check and silently discard every claim for
	// the service. Fail loudly instead.
	if computeUnitsPerRelay == 0 {
		sctx.logger.Error(fmt.Sprintf("compute units per relay for service with ID %q at session start height %d is zero", serviceId, sessionStartHeight))
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf(
			"compute units per relay for service with ID %q at session start height %d is zero (corrupt history entry)",
			serviceId, sessionStartHeight,
		)
	}

	// Store both with the composite key (serviceId + session height). Written together,
	// with no early return in between, so difficulty presence implies cupr presence —
	// which is what GetServiceComputeUnitsPerRelay and the early return above rely on.
	sctx.relayMiningDifficultyMap[cacheKey] = relayMiningDifficulty
	sctx.computeUnitsPerRelayMap[cacheKey] = computeUnitsPerRelay

	return nil
}
