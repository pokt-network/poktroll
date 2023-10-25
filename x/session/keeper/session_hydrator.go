package keeper

import (
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "golang.org/x/crypto/sha3"

	"pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

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
	sessionIdBz []byte
}

func NewSessionHydrator(
	appAddress string,
	serviceId string,
	blockHeight int64,
) *sessionHydrator {
	sessionHeader := &types.SessionHeader{
		ApplicationAddress: appAddress,
		ServiceId:          &sharedtypes.ServiceId{Id: serviceId},
	}
	return &sessionHydrator{
		sessionHeader: sessionHeader,
		session:       &types.Session{},
		blockHeight:   blockHeight,
		sessionIdBz:   make([]byte, 0),
	}
}

// GetSession implements of the exposed `UtilityModule.GetSession` function
// TECHDEBT(#519): Add custom error types depending on the type of issue that occurred and assert on them in the unit tests.
func (k Keeper) HydrateSession(ctx sdk.Context, sh *sessionHydrator) (*types.Session, error) {
	logger := k.Logger(ctx).With("method", "hydrateSession")

	if err := k.hydrateSessionMetadata(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate the session metadata: %v", err)
	}
	logger.Debug("Finished hydrating session metadata")

	if err := k.hydrateSessionID(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate the session ID: %v", err)
	}
	logger.Info("Finished hydrating session ID: %s", sh.sessionHeader.SessionId)

	if err := k.hydrateSessionApplication(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate application for session: %v", err)
	}
	logger.Debug("Finished hydrating session application: %+v", sh.session.Application)

	if err := k.hydrateSessionSuppliers(ctx, sh); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate suppliers for session: %v", err)
	}
	logger.Debug("Finished hydrating session suppliers: %+v")

	sh.session.Header = sh.sessionHeader
	sh.session.SessionId = sh.sessionHeader.SessionId

	return sh.session, nil
}

// hydrateSessionMetadata hydrates metadata related to the session such as the height at which the session started, its number, the number of blocks per session, etc..
func (k Keeper) hydrateSessionMetadata(ctx sdk.Context, sh *sessionHydrator) error {
	sh.session.NumBlocksPerSession = NumBlocksPerSession
	sh.session.SessionNumber = int64(sh.blockHeight/NumBlocksPerSession) + 1
	sh.sessionHeader.SessionStartBlockHeight = sh.blockHeight - (sh.blockHeight % NumBlocksPerSession)
	return nil
}

// hydrateSessionID use both session and on-chain data to determine a unique session ID
func (k Keeper) hydrateSessionID(ctx sdk.Context, sh *sessionHydrator) error {
	// TODO_TECHDEBT: Need to retrieve the block hash at SessionStartBlockHeight, NOT THE CURRENT ONE
	prevHashBz := ctx.HeaderHash()
	appPubKeyBz := []byte(sh.sessionHeader.ApplicationAddress)
	serviceIdBz := []byte(sh.sessionHeader.ServiceId.Id)
	sessionHeightBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionHeightBz, uint64(sh.sessionHeader.SessionStartBlockHeight))

	sh.sessionIdBz = concatWithDelimiter(SessionIDComponentDelimiter, prevHashBz, serviceIdBz, appPubKeyBz, sessionHeightBz)
	sh.sessionHeader.SessionId = hex.EncodeToString(shas3Hash(sh.sessionIdBz))

	return nil
}

// hydrateSessionApplication hydrates the full Application actor based on the address provided
func (k Keeper) hydrateSessionApplication(ctx sdk.Context, sh *sessionHydrator) error {
	app, appIsFound := k.appKeeper.GetApplication(ctx, sh.sessionHeader.ApplicationAddress)
	if !appIsFound {
		return sdkerrors.Wrapf(types.ErrHydratingSession, "failed to find session application")
	}
	sh.session.Application = &app
	return nil
}

// hydrateSessionSuppliers finds the suppliers that are staked at the session height and populates the session with them
func (k Keeper) hydrateSessionSuppliers(ctx sdk.Context, sh *sessionHydrator) error {
	logger := k.Logger(ctx).With("method", "hydrateSessionSuppliers")

	// TECHDEBT(@Olshansk): Need to retrieve the suppliers at SessionStartBlockHeight, NOT THE CURRENT ONE
	// retrieve the suppliers at the current block height
	suppliers := k.supplierKeeper.GetAllSupplier(ctx)

	candidateSuppliers := make([]*sharedtypes.Supplier, 0)
	for _, supplier := range suppliers {
		// OPTIMIZE: If `supplier.Services` was a map[string]struct{}, we could eliminate `slices.Contains()`'s loop
		for _, supplierServiceConfig := range supplier.Services {
			if supplierServiceConfig.ServiceId.Id == sh.sessionHeader.ServiceId.Id {
				candidateSuppliers = append(candidateSuppliers, &supplier)
				break
			}
		}
	}

	if len(candidateSuppliers) < NumSupplierPerSession {
		logger.Info("number of available suppliers (%d) is less than the number of suppliers per session (%d)", len(candidateSuppliers), NumSupplierPerSession)
		sh.session.Suppliers = candidateSuppliers
	} else {
		sh.session.Suppliers = pseudoRandomSelection(candidateSuppliers, NumSupplierPerSession, sh.sessionIdBz)
	}

	return nil
}

// TODO_INVESTIGATE: We are using a `Go` native implementation for a pseudo-random number generator. In order
// for it to be language agnostic, a general purpose algorithm MUST be used.
// pseudoRandomSelection returns a random subset of the candidates.
func pseudoRandomSelection(candidates []*sharedtypes.Supplier, numTarget int, sessionIdBz []byte) []*sharedtypes.Supplier {
	// Take the first 8 bytes of sessionId to use as the seed
	// NB: There is specific reason why `BigEndian` was chosen over `LittleEndian` in this specific context.
	seed := int64(binary.BigEndian.Uint64(shas3Hash(sessionIdBz)[:8]))

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
// NB: A map pointing to empty structs is used to simulate set behaviour.
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

func shas3Hash(bz []byte) []byte {
	hasher := crypto.SHA3_256.New()
	hasher.Write(bz)
	return hasher.Sum(nil)
}
