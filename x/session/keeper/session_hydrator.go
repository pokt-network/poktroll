package keeper

import (
	"bytes"
	"context"
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "golang.org/x/crypto/sha3"

	"github.com/pokt-network/poktroll/telemetry"
	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var SHA3HashLen = crypto.SHA3_256.Size()

const (
	sessionIDComponentDelimiter = "."
)

type sessionHydrator struct {
	// The session header that is used to hydrate the rest of the session data
	sessionHeader *types.SessionHeader

	// The fully hydrated session object
	session *types.Session

	// The height at which the session being request
	blockHeight int64

	// A redundant helper that maintains a hex decoded copy of `session.Id` used for session hydration
	sessionIDBz []byte
}

func NewSessionHydrator(
	appAddress string,
	serviceId string,
	blockHeight int64,
) *sessionHydrator {
	sessionHeader := &types.SessionHeader{
		ApplicationAddress: appAddress,
		ServiceId:          serviceId,
	}
	return &sessionHydrator{
		sessionHeader: sessionHeader,
		session:       &types.Session{},
		blockHeight:   blockHeight,
		sessionIDBz:   make([]byte, 0),
	}
}

// GetSession implements of the exposed `UtilityModule.GetSession` function
// TECHDEBT(#519,#348): Add custom error types depending on the type of issue that occurred and assert on them in the unit tests.
func (k Keeper) HydrateSession(ctx context.Context, sh *sessionHydrator) (*types.Session, error) {
	logger := k.Logger().With("method", "hydrateSession")

	if err := k.hydrateSessionMetadata(ctx, sh); err != nil {
		return nil, err
	}
	logger.Debug("Finished hydrating session metadata")

	if err := k.hydrateSessionID(ctx, sh); err != nil {
		return nil, err
	}
	logger.Debug(fmt.Sprintf("Finished hydrating session ID: %s", sh.sessionHeader.SessionId))

	if err := k.hydrateSessionApplication(ctx, sh); err != nil {
		return nil, err
	}
	logger.Debug(fmt.Sprintf("Finished hydrating session application: %+v", sh.session.Application))

	if err := k.hydrateSessionSuppliers(ctx, sh); err != nil {
		return nil, err
	}
	logger.Debug("Finished hydrating session suppliers")

	sh.session.Header = sh.sessionHeader
	sh.session.SessionId = sh.sessionHeader.SessionId

	return sh.session, nil
}

// hydrateSessionMetadata hydrates metadata related to the session such as the height at which the session started, its number, the number of blocks per session, etc..
func (k Keeper) hydrateSessionMetadata(ctx context.Context, sh *sessionHydrator) error {
	// TODO_TEST: Add a test if `blockHeight` is ahead of the current chain or what this node is aware of

	lastCommittedBlockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if sh.blockHeight > lastCommittedBlockHeight {
		return types.ErrSessionHydration.Wrapf(
			"block height %d is ahead of the last committed block height %d",
			sh.blockHeight, lastCommittedBlockHeight,
		)
	}

	// TODO_MAINNET_MIGRATION(@red-0ne, #543): If the num_blocks_per_session param
	// has ever been changed, this function may cause unexpected behavior for historical sessions.
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sh.session.NumBlocksPerSession = int64(sharedParams.NumBlocksPerSession)
	sh.session.SessionNumber = sharedtypes.GetSessionNumber(&sharedParams, sh.blockHeight)

	sh.sessionHeader.SessionStartBlockHeight = sharedtypes.GetSessionStartHeight(&sharedParams, sh.blockHeight)
	sh.sessionHeader.SessionEndBlockHeight = sharedtypes.GetSessionEndHeight(&sharedParams, sh.blockHeight)
	return nil
}

// hydrateSessionID use both session and onchain data to determine a unique session ID
func (k Keeper) hydrateSessionID(ctx context.Context, sh *sessionHydrator) error {
	prevHashBz := k.GetBlockHash(ctx, sh.sessionHeader.SessionStartBlockHeight)

	// TODO_TECHDEBT: In the future, we will need to validate that the Service is
	// a valid service depending on whether or not its permissioned or permissionless

	if err := sharedtypes.IsValidServiceId(sh.sessionHeader.ServiceId); err != nil {
		return types.ErrSessionHydration.Wrapf("%v", err.Error())
	}

	sh.sessionHeader.SessionId, sh.sessionIDBz = k.GetSessionId(
		ctx,
		sh.sessionHeader.ApplicationAddress,
		sh.sessionHeader.ServiceId,
		prevHashBz,
		sh.blockHeight,
	)

	return nil
}

// hydrateSessionApplication hydrates the full Application actor based on the address provided
func (k Keeper) hydrateSessionApplication(ctx context.Context, sh *sessionHydrator) error {
	foundApp, isAppFound := k.applicationKeeper.GetApplication(ctx, sh.sessionHeader.ApplicationAddress)
	if !isAppFound {
		return types.ErrSessionAppNotFound.Wrapf(
			"could not find app with address %q at height %d",
			sh.sessionHeader.ApplicationAddress,
			sh.sessionHeader.SessionStartBlockHeight,
		)
	}

	// Do not provide sessions for applications that initiated the unstaking process
	// and that are no longer active.
	if !foundApp.IsActive(sh.sessionHeader.SessionEndBlockHeight) {
		return types.ErrSessionAppNotActive.Wrapf(
			"application %q is not active for session %s",
			sh.sessionHeader.ApplicationAddress,
			sh.sessionHeader.SessionId,
		)
	}

	for _, appServiceConfig := range foundApp.ServiceConfigs {
		if appServiceConfig.ServiceId == sh.sessionHeader.ServiceId {
			sh.session.Application = &foundApp
			return nil
		}
	}

	return types.ErrSessionAppNotStakedForService.Wrapf(
		"application %q not staked for service ID %q",
		sh.sessionHeader.ApplicationAddress,
		sh.sessionHeader.ServiceId,
	)
}

// hydrateSessionSuppliers finds the suppliers that are staked at the session
// height and populates the session with them.
func (k Keeper) hydrateSessionSuppliers(ctx context.Context, sh *sessionHydrator) error {
	logger := k.Logger().With("method", "hydrateSessionSuppliers")

	// TODO_MAINNET: Use the number of suppliers per session used at query height (i.e. sh.blockHeight).
	// Currently, the session is hydrated with the "current" (i.e. latest block)
	// NumSuppliersPerSession param value.
	// We need to account for the value at query height to ensure:
	// - The session is hydrated with the correct historical number of suppliers
	// - Changes between query height and current height are properly handled
	// Refer to the following discussion for more details:
	// https://github.com/pokt-network/poktroll/pull/1103#discussion_r1992214953
	numSuppliersPerSession := int(k.GetParams(ctx).NumSuppliersPerSession)

	// Map supplier operator addresses to random weights for deterministic sorting.
	// This ensures fair distribution when:
	// - NumCandidateSuppliers exceeds NumSuppliersPerSession
	// - We need to randomly but fairly determine which suppliers can serve Applications
	candidatesToRandomWeight := make(map[string]int)
	candidateSupplierConfigs := make([]*sharedtypes.ServiceConfigUpdate, 0)

	// Get all suppliers without service ID filtering because:
	// - Suppliers may not be active for the session's service ID at "query height"
	// - We cannot filter by supplier.Services which only represents current (i.e. latest) height
	sessionSupplierServiceConfigIterator := k.supplierKeeper.GetServiceConfigUpdatesIterator(
		ctx,
		sh.sessionHeader.ServiceId,
	)
	defer sessionSupplierServiceConfigIterator.Close()

	for ; sessionSupplierServiceConfigIterator.Valid(); sessionSupplierServiceConfigIterator.Next() {
		supplierServiceConfigUpdate, err := sessionSupplierServiceConfigIterator.Value()
		if err != nil {
			logger.Error(fmt.Sprintf("could not get supplier service config from iterator: %v", err))
			return err
		}

		// Check if supplier is authorized to serve this service at query block height.
		if supplierServiceConfigUpdate.IsActive(sh.blockHeight) {
			candidateSupplierConfigs = append(candidateSupplierConfigs, supplierServiceConfigUpdate)
		}
	}

	defer telemetry.SessionSuppliersGauge(len(candidateSupplierConfigs), numSuppliersPerSession, sh.sessionHeader.ServiceId)

	if len(candidateSupplierConfigs) == 0 {
		logger.Error("[ERROR] no suppliers found for session")
		return types.ErrSessionSuppliersNotFound.Wrapf(
			"could not find suppliers for service %s at height %d",
			sh.sessionHeader.ServiceId,
			sh.sessionHeader.SessionStartBlockHeight,
		)
	}

	// If the number of available suppliers is less than the maximum number of
	// possible suppliers per session, use all available suppliers.
	if len(candidateSupplierConfigs) < numSuppliersPerSession {
		logger.Debug(fmt.Sprintf(
			"Number of available suppliers (%d) is less than the maximum number of possible suppliers per session (%d)",
			len(candidateSupplierConfigs),
			numSuppliersPerSession,
		))
		suppliers := k.getServiceConfigsSuppliers(ctx, candidateSupplierConfigs)
		sh.session.Suppliers = suppliers

		return nil
	}

	for _, serviceConfigUpdate := range candidateSupplierConfigs {
		supplierOperatorAddress := serviceConfigUpdate.OperatorAddress
		candidatesToRandomWeight[supplierOperatorAddress] = generateSupplierRandomWeight(supplierOperatorAddress, sh.sessionIDBz)
	}

	sortedCandidates := sortCandidateSupplierConfigsBySupplierWeight(candidateSupplierConfigs, candidatesToRandomWeight)
	suppliers := k.getServiceConfigsSuppliers(ctx, sortedCandidates[:numSuppliersPerSession])
	sh.session.Suppliers = suppliers

	return nil
}

// getServiceConfigsSuppliers retrieves Supplier objects for the given service config updates.
// It takes a list of service configuration updates and resolves them to their corresponding
// supplier objects.
//
// For each service config update it:
// - Fetches the corresponding dehydrated supplier from the supplier keeper
// - Attaches only the relevant service configuration
// - Includes the supplier in the returned list.
func (k Keeper) getServiceConfigsSuppliers(
	ctx context.Context,
	candidateServiceConfigUpdates []*sharedtypes.ServiceConfigUpdate,
) []*sharedtypes.Supplier {
	suppliers := make([]*sharedtypes.Supplier, 0)
	for _, serviceConfigUpdate := range candidateServiceConfigUpdates {
		supplier, found := k.supplierKeeper.GetDehydratedSupplier(ctx, serviceConfigUpdate.OperatorAddress)
		if !found {
			continue
		}
		supplier.Services = []*sharedtypes.SupplierServiceConfig{serviceConfigUpdate.Service}
		suppliers = append(suppliers, &supplier)
	}
	return suppliers
}

// Generate deterministic random weight for supplier:
// 1. Combine session ID and supplier's operator address to create unique seed
// 2. Hash the seed using SHA3-256
// 3. Take first 8 bytes of hash as random weight
func generateSupplierRandomWeight(supplierOperatorAddress string, sessionIDBz []byte) int {
	candidateSeed := concatWithDelimiter(
		sessionIDComponentDelimiter,
		sessionIDBz,
		[]byte(supplierOperatorAddress),
	)
	candidateSeedHash := sha3Hash(candidateSeed)
	return int(binary.BigEndian.Uint64(candidateSeedHash[:8]))
}

func concatWithDelimiter(delimiter string, bz ...[]byte) []byte {
	return bytes.Join(bz, []byte(delimiter))
}

func sha3Hash(bz []byte) []byte {
	hasher := crypto.SHA3_256.New()
	hasher.Write(bz)
	return hasher.Sum(nil)
}

// GetSessionId returns the string and bytes representation of the sessionId
// given the application address, service ID, block hash, and block height
// that is used to get the session start block height.
func (k Keeper) GetSessionId(
	ctx context.Context,
	appAddr,
	serviceId string,
	blockHashBz []byte,
	blockHeight int64,
) (sessionId string, sessionIdBz []byte) {
	sharedParams := k.sharedKeeper.GetParams(ctx)
	return GetSessionId(&sharedParams, appAddr, serviceId, blockHashBz, blockHeight)
}

// GetSessionId returns the string and bytes representation of the sessionId for the
// session containing blockHeight, given the shared onchain parameters, application
// address, service ID, and block hash.
func GetSessionId(
	sharedParams *sharedtypes.Params,
	appAddr,
	serviceId string,
	blockHashBz []byte,
	blockHeight int64,
) (sessionId string, sessionIdBz []byte) {
	appAddrBz := []byte(appAddr)
	serviceIdBz := []byte(serviceId)

	sessionStartHeightBz := getSessionStartBlockHeightBz(sharedParams, blockHeight)
	sessionIdBz = concatWithDelimiter(
		sessionIDComponentDelimiter,
		blockHashBz,
		serviceIdBz,
		appAddrBz,
		sessionStartHeightBz,
	)
	sessionId = hex.EncodeToString(sha3Hash(sessionIdBz))

	return sessionId, sessionIdBz
}

// getSessionStartBlockHeightBz returns the bytes representation of the session
// start height for the session containing blockHeight, given the shared onchain
// parameters.
func getSessionStartBlockHeightBz(sharedParams *sharedtypes.Params, blockHeight int64) []byte {
	sessionStartBlockHeight := sharedtypes.GetSessionStartHeight(sharedParams, blockHeight)
	sessionStartBlockHeightBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionStartBlockHeightBz, uint64(sessionStartBlockHeight))
	return sessionStartBlockHeightBz
}

// sortCandidateSupplierConfigsBySupplierWeight sorts the given service config
// list by their corresponding suppliers addresses using the provided random weights map.
func sortCandidateSupplierConfigsBySupplierWeight(
	candidateSuppliers []*sharedtypes.ServiceConfigUpdate,
	candidatesToRandomWeight map[string]int,
) []*sharedtypes.ServiceConfigUpdate {
	// Sort suppliers operator addresses deterministically based on their random weights.
	// If weights are equal, sort by operator address to ensure consistent ordering.
	weightedSupplierSortFn := func(serviceConfigUpdateA, serviceConfigUpdateB *sharedtypes.ServiceConfigUpdate) int {
		// Get the pre-calculated random weights for both suppliers
		weightA := candidatesToRandomWeight[serviceConfigUpdateA.OperatorAddress]
		weightB := candidatesToRandomWeight[serviceConfigUpdateB.OperatorAddress]

		// Calculate the difference between weights.
		weightDiff := weightA - weightB

		// If weights are equal, use operator addresses as a tiebreaker
		// to ensure deterministic ordering.
		if weightDiff == 0 {
			return bytes.Compare(
				[]byte(serviceConfigUpdateA.OperatorAddress),
				[]byte(serviceConfigUpdateB.OperatorAddress),
			)
		}

		// Sort based on weight difference
		return weightDiff
	}

	slices.SortFunc(candidateSuppliers, weightedSupplierSortFn)
	return candidateSuppliers
}
