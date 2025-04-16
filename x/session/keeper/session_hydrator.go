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

	if !sharedtypes.IsValidServiceId(sh.sessionHeader.ServiceId) {
		return types.ErrSessionHydration.Wrapf("invalid service ID: %s", sh.sessionHeader.ServiceId)
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

	// Get all suppliers without service ID filtering because:
	// - Suppliers may not be active for the session's service ID at "query height"
	// - We cannot filter by supplier.Services which only represents current (i.e. latest) height
	suppliers := k.supplierKeeper.GetAllSuppliers(ctx)

	// Map supplier operator addresses to random weights for deterministic sorting.
	// This ensures fair distribution when:
	// - NumCandidateSuppliers exceeds NumSuppliersPerSession
	// - We need to randomly but fairly determine which suppliers can serve Applications
	candidatesToRandomWeight := make(map[string]int)
	candidateSuppliers := make([]*sharedtypes.Supplier, 0)

	for _, supplier := range suppliers {
		// Check if supplier is authorized to serve this service at query block height.
		if supplier.IsActive(uint64(sh.blockHeight), sh.sessionHeader.ServiceId) {
			// DEV_NOTE: Performance optimization for session data:
			// - Suppliers often have multiple service configs for various services
			// - Include only the service config relevant to this specific session
			// - Remove all other service configs from the Supplier object to reduce its size
			// - Minimize data transfer overhead when sending sessions over the network

			// Do not check if sessionServiceConfigIdx is -1 since IsActive is already doing that.
			sessionServiceConfigIdx := getSupplierSessionServiceConfigIdx(&supplier, sh.sessionHeader.ServiceId)

			// TODO_POST_MAINNET: Have dedicated proto type for session suppliers hydration.
			// * Create a distinct supplier proto type specific for session hydration
			// * Avoid confusion between full supplier records and session supplier records
			// * Include only data relevant to the session:
			//   - OperatorAddress
			//   - Stake
			//   - SessionServiceConfig
			supplier.Services = supplier.Services[:sessionServiceConfigIdx+1]
			supplier.ServiceConfigHistory = nil

			candidateSuppliers = append(candidateSuppliers, &supplier)
		}
	}

	defer telemetry.SessionSuppliersGauge(len(candidateSuppliers), numSuppliersPerSession, sh.sessionHeader.ServiceId)

	if len(candidateSuppliers) == 0 {
		logger.Error("[ERROR] no suppliers found for session")
		return types.ErrSessionSuppliersNotFound.Wrapf(
			"could not find suppliers for service %s at height %d",
			sh.sessionHeader.ServiceId,
			sh.sessionHeader.SessionStartBlockHeight,
		)
	}

	// If the number of available suppliers is less than the maximum number of
	// possible suppliers per session, use all available suppliers.
	if len(candidateSuppliers) < numSuppliersPerSession {
		logger.Debug(fmt.Sprintf(
			"Number of available suppliers (%d) is less than the maximum number of possible suppliers per session (%d)",
			len(candidateSuppliers),
			numSuppliersPerSession,
		))
		sh.session.Suppliers = candidateSuppliers

		return nil
	}

	for _, supplier := range candidateSuppliers {
		candidatesToRandomWeight[supplier.OperatorAddress] = generateSupplierRandomWeight(supplier, sh.sessionIDBz)
	}

	sortedCandidates := sortCandidateSuppliersByHeight(candidateSuppliers, candidatesToRandomWeight)
	sh.session.Suppliers = sortedCandidates[:numSuppliersPerSession]

	return nil
}

// Generate deterministic random weight for supplier:
// 1. Combine session ID and supplier's operator address to create unique seed
// 2. Hash the seed using SHA3-256
// 3. Take first 8 bytes of hash as random weight
func generateSupplierRandomWeight(supplier *sharedtypes.Supplier, sessionIDBz []byte) int {
	candidateSeed := concatWithDelimiter(
		sessionIDComponentDelimiter,
		sessionIDBz,
		[]byte(supplier.OperatorAddress),
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

// sortCandidateSuppliersByHeight sorts the given supplier list by the provided
// random weights map.
func sortCandidateSuppliersByHeight(
	candidateSuppliers []*sharedtypes.Supplier,
	candidatesToRandomWeight map[string]int,
) []*sharedtypes.Supplier {
	// Sort suppliers deterministically based on their random weights.
	// If weights are equal, sort by operator address to ensure consistent ordering.
	weightedSupplierSortFn := func(supplierA, supplierB *sharedtypes.Supplier) int {
		// Get the pre-calculated random weights for both suppliers
		weightA := candidatesToRandomWeight[supplierA.OperatorAddress]
		weightB := candidatesToRandomWeight[supplierB.OperatorAddress]

		// Calculate the difference between weights.
		weightDiff := weightA - weightB

		// If weights are equal, use operator addresses as a tiebreaker
		// to ensure deterministic ordering.
		if weightDiff == 0 {
			return bytes.Compare([]byte(supplierA.OperatorAddress), []byte(supplierB.OperatorAddress))
		}

		// Sort based on weight difference
		return weightDiff
	}

	slices.SortFunc(candidateSuppliers, weightedSupplierSortFn)
	return candidateSuppliers
}

// getSupplierSessionServiceConfigIdx returns the index of the session service
// config for the given service ID in the supplier's service config.
func getSupplierSessionServiceConfigIdx(
	supplier *sharedtypes.Supplier,
	serviceId string,
) int {
	return slices.IndexFunc(
		supplier.Services,
		func(s *sharedtypes.SupplierServiceConfig) bool {
			return s.ServiceId == serviceId
		},
	)
}
