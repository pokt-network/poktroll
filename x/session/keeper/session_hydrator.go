package keeper

import (
	"context"
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "golang.org/x/crypto/sha3"

	"github.com/pokt-network/poktroll/x/session/types"
	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var SHA3HashLen = crypto.SHA3_256.Size()

// TODO_BLOCKER(#21): Make these configurable governance param
const (
	// TODO_BLOCKER: Remove direct usage of these constants in helper functions
	// when they will be replaced by governance params
	NumBlocksPerSession = 4
	// Duration of the grace period in number of sessions
	SessionGracePeriod          = 1
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
func (k Keeper) HydrateSession(ctx context.Context, sh *sessionHydrator) (*types.Session, error) {
	logger := k.Logger().With("method", "hydrateSession")

	if err := k.hydrateSessionMetadata(ctx, sh); err != nil {
		return nil, err
	}
	logger.Debug("Finished hydrating session metadata")

	if err := k.hydrateSessionID(ctx, sh); err != nil {
		return nil, err
	}
	logger.Info(fmt.Sprintf("Finished hydrating session ID: %s", sh.sessionHeader.SessionId))

	if err := k.hydrateSessionApplication(ctx, sh); err != nil {
		return nil, err
	}
	logger.Debug("Finished hydrating session application: %+v", sh.session.Application)

	if err := k.hydrateSessionSuppliers(ctx, sh); err != nil {
		return nil, err
	}
	logger.Debug("Finished hydrating session suppliers: %+v")

	sh.session.Header = sh.sessionHeader
	sh.session.SessionId = sh.sessionHeader.SessionId

	return sh.session, nil
}

// hydrateSessionMetadata hydrates metadata related to the session such as the height at which the session started, its number, the number of blocks per session, etc..
func (k Keeper) hydrateSessionMetadata(ctx context.Context, sh *sessionHydrator) error {
	// TODO_TECHDEBT: Add a test if `blockHeight` is ahead of the current chain or what this node is aware of

	lastCommittedBlockHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if sh.blockHeight > lastCommittedBlockHeight {
		return types.ErrSessionHydration.Wrapf(
			"block height %d is ahead of the last committed block height %d",
			sh.blockHeight, lastCommittedBlockHeight,
		)
	}

	sh.session.NumBlocksPerSession = NumBlocksPerSession
	sh.session.SessionNumber = GetSessionNumber(sh.blockHeight)

	sh.sessionHeader.SessionStartBlockHeight = GetSessionStartBlockHeight(sh.blockHeight)
	sh.sessionHeader.SessionEndBlockHeight = GetSessionEndBlockHeight(sh.blockHeight)
	return nil
}

// hydrateSessionID use both session and on-chain data to determine a unique session ID
func (k Keeper) hydrateSessionID(ctx context.Context, sh *sessionHydrator) error {
	prevHashBz := k.GetBlockHash(ctx, sh.sessionHeader.SessionStartBlockHeight)

	// TODO_TECHDEBT: In the future, we will need to validate that the Service is a valid service depending on whether
	// or not its permissioned or permissionless

	if !sharedhelpers.IsValidService(sh.sessionHeader.Service) {
		return types.ErrSessionHydration.Wrapf("invalid service: %v", sh.sessionHeader.Service)
	}

	sh.sessionHeader.SessionId, sh.sessionIDBz = GetSessionId(
		sh.sessionHeader.ApplicationAddress,
		sh.sessionHeader.Service.Id,
		prevHashBz,
		sh.blockHeight,
	)

	return nil
}

// hydrateSessionApplication hydrates the full Application actor based on the address provided
func (k Keeper) hydrateSessionApplication(ctx context.Context, sh *sessionHydrator) error {
	foundApp, appIsFound := k.applicationKeeper.GetApplication(ctx, sh.sessionHeader.ApplicationAddress)
	if !appIsFound {
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

	// TODO_TECHDEBT(@Olshansk, @bryanchriswhite): Need to retrieve the suppliers at SessionStartBlockHeight,
	// NOT THE CURRENT ONE which is what's provided by the context. For now, for simplicity,
	// only retrieving the suppliers at the current block height which could create a discrepancy
	// if new suppliers were staked mid session.
	// TODO(@bryanchriswhite): Investigate if `BlockClient` + `ReplayObservable` where `N = SessionLength` could be used here.`
	suppliers := k.supplierKeeper.GetAllSuppliers(ctx)

	candidateSuppliers := make([]*sharedtypes.Supplier, 0)
	for _, s := range suppliers {
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
			"[WARN] number of available suppliers (%d) is less than the number of suppliers per session (%d)",
			len(candidateSuppliers),
			NumSupplierPerSession,
		))
		sh.session.Suppliers = candidateSuppliers
	} else {
		sh.session.Suppliers = pseudoRandomSelection(candidateSuppliers, NumSupplierPerSession, sh.sessionIDBz)
	}

	return nil
}

// TODO_INVESTIGATE: We are using a `Go` native implementation for a pseudo-random number generator. In order
// for it to be language agnostic, a general purpose algorithm MUST be used.
// pseudoRandomSelection returns a random subset of the candidates.
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

func concatWithDelimiter(delimiter string, bz ...[]byte) (result []byte) {
	for _, b := range bz {
		result = append(result, b...)
		result = append(result, []byte(delimiter)...)
	}
	return result
}

func sha3Hash(bz []byte) []byte {
	hasher := crypto.SHA3_256.New()
	hasher.Write(bz)
	return hasher.Sum(nil)
}

// GetSessionStartBlockHeight returns the block height at which the session starts
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func GetSessionStartBlockHeight(blockHeight int64) int64 {
	if blockHeight <= 0 {
		return 0
	}

	return blockHeight - ((blockHeight - 1) % NumBlocksPerSession)
}

// GetSessionEndBlockHeight returns the block height at which the session ends
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetSessionEndBlockHeight(blockHeight int64) int64 {
	if blockHeight <= 0 {
		return 0
	}

	return GetSessionStartBlockHeight(blockHeight) + NumBlocksPerSession - 1
}

// GetSessionNumber returns the session number given the block height.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func GetSessionNumber(blockHeight int64) int64 {
	if blockHeight <= 0 {
		return 0
	}

	return ((blockHeight - 1) / NumBlocksPerSession) + 1
}

// GetSessionId returns the string and bytes representation of the sessionId
// given the application public key, service ID, block hash, and block height
// that is used to get the session start block height.
func GetSessionId(
	appPubKey,
	serviceId string,
	blockHashBz []byte,
	blockHeight int64,
) (sessionId string, sessionIdBz []byte) {
	appPubKeyBz := []byte(appPubKey)
	serviceIdBz := []byte(serviceId)

	blockHeightBz := getSessionStartBlockHeightBz(blockHeight)
	sessionIdBz = concatWithDelimiter(
		SessionIDComponentDelimiter,
		blockHashBz,
		serviceIdBz,
		appPubKeyBz,
		blockHeightBz,
	)
	sessionId = hex.EncodeToString(sha3Hash(sessionIdBz))

	return sessionId, sessionIdBz
}

// GetSessionGracePeriodBlockCount returns the number of blocks in the session
// grace period.
func GetSessionGracePeriodBlockCount() int64 {
	return SessionGracePeriod * NumBlocksPerSession
}

// getSessionStartBlockHeightBz returns the bytes representation of the session
// start block height given the block height.
func getSessionStartBlockHeightBz(blockHeight int64) []byte {
	sessionStartBlockHeight := GetSessionStartBlockHeight(blockHeight)
	sessionStartBlockHeightBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionStartBlockHeightBz, uint64(sessionStartBlockHeight))
	return sessionStartBlockHeightBz
}
