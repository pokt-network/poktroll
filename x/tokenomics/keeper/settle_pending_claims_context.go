package keeper

import (
	"context"
	"fmt"
	"slices"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicekeeper "github.com/pokt-network/poktroll/x/service/keeper"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

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

	// Cache of parameters used during the settlement process to prevent repeated KV store lookups.
	sharedParams     sharedtypes.Params
	tokenomicsParams tokenomicstypes.Params
}

// NewSettlementContext creates a new settlement context with all necessary caches initialized.
// This function establishes an efficient memory structure that prevents redundant KV store operations
// during claim processing, improving settlement performance for high volume operations.
func NewSettlementContext(
	ctx context.Context,
	tokenomicsKeeper *Keeper,
	logger cosmoslog.Logger,
) *settlementContext {
	return &settlementContext{
		keeper: tokenomicsKeeper,
		logger: logger.With("module", "settlement_context"),

		supplierMap:      make(map[string]int),
		settledSuppliers: make([]*sharedtypes.Supplier, 0),

		applicationMap:             make(map[string]int),
		settledApplications:        make([]*apptypes.Application, 0),
		applicationInitialStakeMap: make(map[string]cosmostypes.Coin),

		serviceMap:               make(map[string]*sharedtypes.Service),
		relayMiningDifficultyMap: make(map[string]servicetypes.RelayMiningDifficulty),
		sharedParams:             tokenomicsKeeper.sharedKeeper.GetParams(ctx),
		tokenomicsParams:         tokenomicsKeeper.GetParams(ctx),
	}
}

// FlushAllActorsToStore batch save all accumulated applications and suppliers to the store.
// It is intended to be called after all claims have been processed.

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
		return tokenomicstypes.ErrTokenomicsSessionHeaderNil
	}

	// Cache service and difficulty
	serviceId := claim.SessionHeader.ServiceId
	if err := sctx.cacheServiceAndDifficulty(ctx, serviceId); err != nil {
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

// GetRelayMiningDifficulty retrieves the cached relay mining difficulty for a specific service.
func (sctx *settlementContext) GetRelayMiningDifficulty(serviceId string) (servicetypes.RelayMiningDifficulty, error) {
	if relayMiningDifficulty, ok := sctx.relayMiningDifficultyMap[serviceId]; ok {
		return relayMiningDifficulty, nil
	}

	sctx.logger.Error(fmt.Sprintf("relay mining difficulty for service with ID %q not found", serviceId))
	return servicetypes.RelayMiningDifficulty{}, tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf(
		"relay mining difficulty for service with ID %q not found", serviceId,
	)
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
	if idx, ok := sctx.supplierMap[supplierOperatorAddress]; ok {
		cachedSupplier := sctx.settledSuppliers[idx]

		// Supplier is cached, ensure that it has a service configuration corresponding
		// to the claim's service ID.
		sctx.cacheSupplierServiceConfig(ctx, cachedSupplier, serviceId)

		return nil // Supplier already cached
	}

	// Retrieve the onchain staked dehydrated supplier record since other service
	// configurations are not needed for the claim settlement.
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
		return nil // Application already cached
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

// cacheServiceAndDifficulty ensures the service and its relay mining difficulty are cached in the settlement context.
//
// This prevents repeated KV store lookups across multiple claims targeting the same service.
func (sctx *settlementContext) cacheServiceAndDifficulty(ctx context.Context, serviceId string) error {
	if _, ok := sctx.serviceMap[serviceId]; ok {
		return nil // Service already cached
	}

	// Retrieve the service record
	service, isServiceFound := sctx.keeper.serviceKeeper.GetService(ctx, serviceId)
	if !isServiceFound {
		return tokenomicstypes.ErrTokenomicsServiceNotFound.Wrapf("service with ID %q not found", serviceId)
	}
	sctx.serviceMap[serviceId] = &service

	// Retrieve or create the relay mining difficulty for the service
	relayMiningDifficulty, found := sctx.keeper.serviceKeeper.GetRelayMiningDifficulty(ctx, service.Id)
	if !found {
		targetNumRelays := sctx.keeper.serviceKeeper.GetParams(ctx).TargetNumRelays
		relayMiningDifficulty = servicekeeper.NewDefaultRelayMiningDifficulty(
			ctx,
			sctx.logger,
			service.Id,
			targetNumRelays,
			targetNumRelays,
		)
	}
	sctx.relayMiningDifficultyMap[service.Id] = relayMiningDifficulty

	return nil
}
