package keeper

import (
	"bytes"
	"context"
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "golang.org/x/crypto/sha3"

	"github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var SHA3HashLen = crypto.SHA3_256.Size()

// TODO_BLOCKER(@bryanchriswhite, #21): Make these configurable governance param
const (
	NumSupplierPerSession       = 15
	SessionIDComponentDelimiter = "."
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
		Service:            &sharedtypes.Service{Id: serviceId},
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
// TODO_BETA: Consider returning an error if the application's stake has become very low.
func (k Keeper) HydrateSession(ctx context.Context, sh *sessionHydrator) (*types.Session, error) {
	logger := k.Logger().With("method", "hydrateSession")

	if err := k.hydrateSessionMetadata(ctx, sh); err != nil {
		return nil, err
	}
	logger.Info("Finished hydrating session metadata")

	if err := k.hydrateSessionID(ctx, sh); err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("Finished hydrating session ID: %s", sh.sessionHeader.SessionId))

	if err := k.hydrateSessionApplication(ctx, sh); err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("Finished hydrating session application: %+v", sh.session.Application))

	if err := k.hydrateSessionSuppliers(ctx, sh); err != nil {
		return nil, err
	}
	logger.Info("Finished hydrating session suppliers")

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

	// TODO_BLOCKER(@bryanchriswhite, #543): If the num_blocks_per_session param has ever been changed,
	// this function may cause unexpected behavior for historical sessions.
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sh.session.NumBlocksPerSession = int64(sharedParams.NumBlocksPerSession)
	sh.session.SessionNumber = shared.GetSessionNumber(&sharedParams, sh.blockHeight)

	sh.sessionHeader.SessionStartBlockHeight = shared.GetSessionStartHeight(&sharedParams, sh.blockHeight)
	sh.sessionHeader.SessionEndBlockHeight = shared.GetSessionEndHeight(&sharedParams, sh.blockHeight)
	return nil
}

// hydrateSessionID use both session and on-chain data to determine a unique session ID
func (k Keeper) hydrateSessionID(ctx context.Context, sh *sessionHydrator) error {
	prevHashBz := k.GetBlockHash(ctx, sh.sessionHeader.SessionStartBlockHeight)

	// TODO_MAINNET: In the future, we will need to validate that the Service is
	// a valid service depending on whether or not its permissioned or permissionless

	if !sharedhelpers.IsValidService(sh.sessionHeader.Service) {
		return types.ErrSessionHydration.Wrapf("invalid service: %v", sh.sessionHeader.Service)
	}

	sh.sessionHeader.SessionId, sh.sessionIDBz = k.GetSessionId(
		ctx,
		sh.sessionHeader.ApplicationAddress,
		sh.sessionHeader.Service.Id,
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

	for _, appServiceConfig := range foundApp.ServiceConfigs {
		if appServiceConfig.Service.Id == sh.sessionHeader.Service.Id {
			sh.session.Application = &foundApp
			return nil
		}
	}

	return types.ErrSessionAppNotStakedForService.Wrapf(
		"application %q not staked for service ID %q",
		sh.sessionHeader.ApplicationAddress,
		sh.sessionHeader.Service.Id,
	)
}

// hydrateSessionSuppliers finds the suppliers that are staked at the session
// height and populates the session with them.
func (k Keeper) hydrateSessionSuppliers(ctx context.Context, sh *sessionHydrator) error {
	logger := k.Logger().With("method", "hydrateSessionSuppliers")

	suppliers := k.supplierKeeper.GetAllSuppliers(ctx)

	candidateSuppliers := make([]*sharedtypes.Supplier, 0)
	for _, s := range suppliers {
		// Exclude suppliers that are inactive (i.e. currently unbonding).
		if !s.IsActive(uint64(sh.sessionHeader.SessionEndBlockHeight), sh.sessionHeader.Service.Id) {
			continue
		}

		// NB: Allocate a new heap variable as s is a value and we're appending
		// to a slice of  pointers; otherwise, we'd be appending new pointers to
		// the same memory address containing the last supplier in the loop.
		supplier := s
		// TODO_OPTIMIZE: If `supplier.Services` was a map[string]struct{}, we could eliminate `slices.Contains()`'s loop
		for _, supplierServiceConfig := range supplier.Services {
			if supplierServiceConfig.Service.Id == sh.sessionHeader.Service.Id {
				candidateSuppliers = append(candidateSuppliers, &supplier)
				break
			}
		}
	}

	if len(candidateSuppliers) == 0 {
		logger.Error("[ERROR] no suppliers found for session")
		return types.ErrSessionSuppliersNotFound.Wrapf(
			"could not find suppliers for service %s at height %d",
			sh.sessionHeader.Service,
			sh.sessionHeader.SessionStartBlockHeight,
		)
	}

	if len(candidateSuppliers) < NumSupplierPerSession {
		logger.Info(fmt.Sprintf(
			"Number of available suppliers (%d) is less than the maximum number of possible suppliers per session (%d)",
			len(candidateSuppliers),
			NumSupplierPerSession,
		))
		sh.session.Suppliers = candidateSuppliers
	} else {
		sh.session.Suppliers = pseudoRandomSelection(candidateSuppliers, NumSupplierPerSession, sh.sessionIDBz)
	}

	return nil
}

// TODO_BETA: We are using a `Go` native implementation for a pseudo-random
// number generator. In order for it to be language agnostic, a general purpose
// algorithm MUST be used. pseudoRandomSelection returns a random subset of the
// candidates.
func pseudoRandomSelection(
	candidates []*sharedtypes.Supplier,
	numTarget int,
	sessionIDBz []byte,
) []*sharedtypes.Supplier {
	// Take the first 8 bytes of sessionId to use as the seed
	// NB: There is specific reason why `BigEndian` was chosen over `LittleEndian` in this specific context.
	seed := int64(binary.BigEndian.Uint64(sha3Hash(sessionIDBz)[:8]))

	// Retrieve the indices for the candidates
	actors := make([]*sharedtypes.Supplier, 0)
	uniqueIndices := uniqueRandomIndices(seed, int64(len(candidates)), int64(numTarget))
	for idx := range uniqueIndices {
		actors = append(actors, candidates[idx])
	}

	return actors
}

// uniqueRandomIndices returns a map of `numIndices` unique random numbers less than `maxIndex`
// seeded by `seed`.
// panics if `numIndicies > maxIndex` since that code path SHOULD never be executed.
// NB: A map pointing to empty structs is used to simulate set behavior.
func uniqueRandomIndices(seed, maxIndex, numIndices int64) map[int64]struct{} {
	// This should never happen
	if numIndices > maxIndex {
		panic(fmt.Sprintf("uniqueRandomIndices: numIndices (%d) is greater than maxIndex (%d)", numIndices, maxIndex))
	}

	// create a new random source with the seed
	randSrc := rand.NewSource(seed)

	// initialize a map to capture the indicesMap we'll return
	indicesMap := make(map[int64]struct{}, maxIndex)

	// The random source could potentially return duplicates, so while loop until we have enough unique indices
	for int64(len(indicesMap)) < numIndices {
		indicesMap[randSrc.Int63()%int64(maxIndex)] = struct{}{}
	}

	return indicesMap
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
// session containing blockHeight, given the shared on-chain parameters, application
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
		SessionIDComponentDelimiter,
		blockHashBz,
		serviceIdBz,
		appAddrBz,
		sessionStartHeightBz,
	)
	sessionId = hex.EncodeToString(sha3Hash(sessionIdBz))

	return sessionId, sessionIdBz
}

// getSessionStartBlockHeightBz returns the bytes representation of the session
// start height for the session containing blockHeight, given the shared on-chain
// parameters.
func getSessionStartBlockHeightBz(sharedParams *sharedtypes.Params, blockHeight int64) []byte {
	sessionStartBlockHeight := shared.GetSessionStartHeight(sharedParams, blockHeight)
	sessionStartBlockHeightBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionStartBlockHeightBz, uint64(sessionStartBlockHeight))
	return sessionStartBlockHeightBz
}
