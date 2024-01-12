package keeper

import (
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "golang.org/x/crypto/sha3" // blank to be able to get the size of the hash

	"github.com/pokt-network/poktroll/x/session/types"
	sharedhelpers "github.com/pokt-network/poktroll/x/shared/helpers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SHA3HashLen is the length of the sha3_256 hash (32 bytes)
var SHA3HashLen = crypto.SHA3_256.Size()

// TODO(#21): Make these configurable governance param
const (
	NumBlocksPerSession         = 4
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

// NewSessionHydrator creates a new session hydrator instance.
func NewSessionHydrator(
	appAddress string,
	serviceID string,
	blockHeight int64,
) *sessionHydrator {
	sessionHeader := &types.SessionHeader{
		ApplicationAddress: appAddress,
		Service:            &sharedtypes.Service{Id: serviceID},
	}
	return &sessionHydrator{
		sessionHeader: sessionHeader,
		session:       &types.Session{},
		blockHeight:   blockHeight,
		sessionIDBz:   make([]byte, 0),
	}
}

// HydrateSession implements of the exposed `UtilityModule.GetSession` function
// TECHDEBT(#519): Add custom error types depending on the type of issue that
// occurred and assert on them in the unit tests.
func (k Keeper) HydrateSession(ctx sdk.Context, sh *sessionHydrator) (*types.Session, error) {
	logger := k.Logger(ctx).With("method", "hydrateSession")

	if err := k.hydrateSessionMetadata(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrSessionHydration, "failed to hydrate the session metadata: %v", err)
	}
	logger.Debug("Finished hydrating session metadata")

	if err := k.hydrateSessionID(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrSessionHydration, "failed to hydrate the session ID: %v", err)
	}
	logger.Info(fmt.Sprintf("Finished hydrating session ID: %s", sh.sessionHeader.SessionId))

	if err := k.hydrateSessionApplication(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrSessionHydration, "failed to hydrate application for session: %v", err)
	}
	logger.Debug("Finished hydrating session application: %+v", sh.session.Application)

	if err := k.hydrateSessionSuppliers(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrSessionHydration, "failed to hydrate suppliers for session: %v", err)
	}
	logger.Debug("Finished hydrating session suppliers: %+v")

	sh.session.Header = sh.sessionHeader
	sh.session.SessionId = sh.sessionHeader.SessionId

	return sh.session, nil
}

// hydrateSessionMetadata hydrates metadata related to the session such as the
// height at which the session started, its number, the number of blocks per
// session, etc..
func (k Keeper) hydrateSessionMetadata(ctx sdk.Context, sh *sessionHydrator) error {
	// TODO_TECHDEBT: Add a test if `blockHeight` is ahead of the current chain or what this node is aware of

	if sh.blockHeight > ctx.BlockHeight() {
		return sdkerrors.Wrapf(
			types.ErrSessionHydration,
			"block height %d is ahead of the current block height %d",
			sh.blockHeight,
			ctx.BlockHeight(),
		)
	}

	sh.session.NumBlocksPerSession = NumBlocksPerSession
	sh.session.SessionNumber = sh.blockHeight / NumBlocksPerSession

	// TODO_BLOCKER: SessionStartBlockHeight should be aligned to NumBlocksPerSession.
	sh.sessionHeader.SessionStartBlockHeight = sh.blockHeight - (sh.blockHeight % NumBlocksPerSession)
	sh.sessionHeader.SessionEndBlockHeight = sh.sessionHeader.SessionStartBlockHeight + NumBlocksPerSession
	return nil
}

// hydrateSessionID use both session and on-chain data to determine a unique session ID
func (k Keeper) hydrateSessionID(_ sdk.Context, sh *sessionHydrator) error {
	// TODO_BLOCKER: Need to retrieve the block hash at SessionStartBlockHeight, but this requires
	// a bit of work and the `ctx` only gives access to the current block/header. See this thread
	// for more details: https://github.com/pokt-network/poktroll/pull/78/files#r1369215667
	// prevHashBz := ctx.HeaderHash()
	prevHash := "TODO_BLOCKER: See the comment above"

	// TODO_TECHDEBT: In the future, we will need to valid that the Service is a valid service depending on whether
	// or not its permissioned or permissionless

	if !sharedhelpers.IsValidService(sh.sessionHeader.Service) {
		return sdkerrors.Wrapf(types.ErrSessionHydration, "invalid service: %v", sh.sessionHeader.Service)
	}

	sh.sessionHeader.SessionId, sh.sessionIDBz = GetSessionId(
		sh.sessionHeader.ApplicationAddress,
		sh.sessionHeader.Service.Id,
		prevHash,
		sh.sessionHeader.SessionStartBlockHeight,
	)

	return nil
}

// hydrateSessionApplication hydrates the full Application actor based on the address provided
func (k Keeper) hydrateSessionApplication(ctx sdk.Context, sh *sessionHydrator) error {
	app, appIsFound := k.appKeeper.GetApplication(ctx, sh.sessionHeader.ApplicationAddress)
	if !appIsFound {
		return sdkerrors.Wrapf(
			types.ErrSessionAppNotFound,
			"could not find app with address: %s at height %d",
			sh.sessionHeader.ApplicationAddress,
			sh.sessionHeader.SessionStartBlockHeight,
		)
	}

	for _, appServiceConfig := range app.ServiceConfigs {
		if appServiceConfig.Service.Id == sh.sessionHeader.Service.Id {
			sh.session.Application = &app
			return nil
		}
	}

	return sdkerrors.Wrapf(
		types.ErrSessionAppNotStakedForService,
		"application %s not staked for service %s",
		sh.sessionHeader.ApplicationAddress,
		sh.sessionHeader.Service.Id,
	)
}

// hydrateSessionSuppliers finds the suppliers that are staked at the session height and populates the session with them
func (k Keeper) hydrateSessionSuppliers(ctx sdk.Context, sh *sessionHydrator) error {
	logger := k.Logger(ctx).With("method", "hydrateSessionSuppliers")

	// TODO_TECHDEBT(@Olshansk, @bryanchriswhite): Need to retrieve the suppliers at SessionStartBlockHeight,
	// NOT THE CURRENT ONE which is what's provided by the context. For now, for simplicity,
	// only retrieving the suppliers at the current block height which could create a discrepancy
	// if new suppliers were staked mid session.
	// TODO(@bryanchriswhite): Investigate if `BlockClient` + `ReplayObservable`
	// where `N = SessionLength` could be used here.`
	suppliers := k.supplierKeeper.GetAllSupplier(ctx)

	candidateSuppliers := make([]*sharedtypes.Supplier, 0)
	for _, s := range suppliers {
		// NB: Allocate a new heap variable as s is a value and we're appending
		// to a slice of  pointers; otherwise, we'd be appending new pointers to
		// the same memory address containing the last supplier in the loop.
		supplier := s
		// TODO_OPTIMIZE: If `supplier.Services` was a map[string]struct{},
		// we could eliminate `slices.Contains()`'s loop
		for _, supplierServiceConfig := range supplier.Services {
			if supplierServiceConfig.Service.Id == sh.sessionHeader.Service.Id {
				candidateSuppliers = append(candidateSuppliers, &supplier)
				break
			}
		}
	}

	if len(candidateSuppliers) == 0 {
		logger.Error(fmt.Sprintf("[ERROR] no suppliers found for session"))
		return sdkerrors.Wrapf(
			types.ErrSessionSuppliersNotFound,
			"could not find suppliers for service %s at height %d",
			sh.sessionHeader.Service,
			sh.sessionHeader.SessionStartBlockHeight,
		)
	}

	if len(candidateSuppliers) < NumSupplierPerSession {
		logger.Info(
			fmt.Sprintf(
				"[WARN] number of available suppliers (%d) is less than the number of suppliers per session (%d)",
				len(candidateSuppliers),
				NumSupplierPerSession,
			),
		)
		sh.session.Suppliers = candidateSuppliers
	} else {
		sh.session.Suppliers = pseudoRandomSelection(candidateSuppliers, NumSupplierPerSession, sh.sessionIDBz)
	}

	return nil
}

// TODO_INVESTIGATE: We are using a `Go` native implementation for a
// pseudo-random number generator. In order for it to be language agnostic, a
// general purpose algorithm MUST be used. pseudoRandomSelection returns a
// random subset of the candidates.
func pseudoRandomSelection(
	candidates []*sharedtypes.Supplier,
	numTarget int,
	sessionIDBz []byte,
) []*sharedtypes.Supplier {
	// Take the first 8 bytes of sessionID to use as the seed
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

func concatWithDelimiter(delimiter string, b ...[]byte) (result []byte) {
	for _, bz := range b {
		result = append(result, bz...)
		result = append(result, []byte(delimiter)...)
	}
	return result
}

func sha3Hash(bz []byte) []byte {
	hasher := crypto.SHA3_256.New()
	hasher.Write(bz)
	return hasher.Sum(nil)
}

// GetSessionId returns the string and bytes representation of the sessionId
// given the application public key, service ID, block hash, and block height.
func GetSessionId(
	appPubKey,
	serviceID,
	blockHash string,
	blockHeight int64,
) (sessionID string, sessionIDBz []byte) {
	appPubKeyBz := []byte(appPubKey)
	serviceIDBz := []byte(serviceID)
	blockHashBz := []byte(blockHash)

	sessionHeightBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionHeightBz, uint64(blockHeight))

	sessionIDBz = concatWithDelimiter(
		SessionIDComponentDelimiter,
		blockHashBz,
		serviceIDBz,
		appPubKeyBz,
		sessionHeightBz,
	)
	sessionID = hex.EncodeToString(sha3Hash(sessionIDBz))

	return sessionID, sessionIDBz
}
