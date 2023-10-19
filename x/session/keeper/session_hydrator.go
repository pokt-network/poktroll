package keeper

import (
	"crypto"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cometbft/cometbft/libs/log"
	cmtlogger "github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	_ "golang.org/x/crypto/sha3"

	"pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

var SHA3HashLen = crypto.SHA3_256.Size()

const (
	// TODO(#XXX): Make these onfigurable governance param
	NumBlocksPerSession   = 4
	NumSupplierPerSession = 15
)

type sessionHydrator struct {
	logger log.Logger

	// The session header that is used to hydrate the reset of the session data
	sessionHeader *types.SessionHeader

	// The fully hydrated session object
	session *types.Session

	// The height at which the session being request
	blockHeight int64

	// A redundant helper that maintains a hex decoded copy of `session.Id` used for session hydration
	sessionIdBz []byte

	// keeper & context related params
	k   *Keeper
	ctx *sdk.Context
}

func NewSessionHydrator(
	logger cmtlogger.Logger,
	appAddress string,
	serviceId *sharedtypes.ServiceId,
	blockHeight int64,
	k Keeper,
	ctx sdk.Context,
) *sessionHydrator {
	shLogger := logger.With("method", "NewSessionHydrator").
		With("appAddr", appAddress).
		With("serviceId.Id", serviceId.Id).
		With("blockHeight", blockHeight)

	sessionHeader := &types.SessionHeader{
		ApplicationAddress: appAddress,
		ServiceId: &sharedtypes.ServiceId{
			Id:   serviceId.Id,
			Name: serviceId.Name,
		},
	}

	return &sessionHydrator{
		logger:        shLogger,
		sessionHeader: sessionHeader,
		session:       &types.Session{},
		blockHeight:   blockHeight,
		sessionIdBz:   make([]byte, 0),
		k:             &k,
		ctx:           &ctx,
	}
}

// GetSession implements of the exposed `UtilityModule.GetSession` function
// TECHDEBT(#519): Add custom error types depending on the type of issue that occurred and assert on them in the unit tests.
func (sh *sessionHydrator) hydrateSession() (*types.Session, error) {
	sh.logger.Info("About to start hydrating the session")

	if err := sh.hydrateSessionMetadata(); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate session metadata: %v", err)
	}
	sh.logger.Debug("Finished hydrating session metadata")

	if err := sh.hydrateSessionID(); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate session ID: %v", err)
	}
	sh.logger.Info("Finished hydrating session ID: %s", sh.sessionHeader.SessionId)

	if err := sh.hydrateSessionApplication(); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate session application: %v", err)
	}
	sh.logger.Debug("Finished hydrating session application: %+v", sh.session.Application)

	if err := sh.hydrateSessionSuppliers(); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrHydratingSession, "failed to hydrate session suppliers: %v", err)
	}
	sh.logger.Debug("Finished hydrating session suppliers: %+v")

	return sh.session, nil
}

// hydrateSessionMetadata hydrates metadata related to the session such as the height at which the session started, its number, the number of blocks per session, etc..
func (sh *sessionHydrator) hydrateSessionMetadata() error {
	sh.session.NumBlocksPerSession = NumBlocksPerSession
	sh.session.SessionNumber = int64(sh.blockHeight / NumBlocksPerSession)
	sh.sessionHeader.SessionStartBlockHeight = sh.blockHeight - (sh.blockHeight % NumBlocksPerSession)
	return nil
}

// hydrateSessionID use both session and on-chain data to determine a unique session ID
func (sh *sessionHydrator) hydrateSessionID() error {
	// TECHDEBT(@Olshansk): Need to retrieve the block hash at SessionStartBlockHeight, NOT THE CURRENT ONE
	prevHashBz := sh.ctx.HeaderHash()
	appPubKeyBz := []byte(sh.sessionHeader.ApplicationAddress)
	serviceIdBz := []byte(sh.sessionHeader.ServiceId.Id)
	sessionHeightBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sessionHeightBz, uint64(sh.sessionHeader.SessionStartBlockHeight))

	sh.sessionIdBz = concat(prevHashBz, serviceIdBz, appPubKeyBz, sessionHeightBz)
	sh.sessionHeader.SessionId = hex.EncodeToString(shas3Hash(sh.sessionIdBz))

	return nil
}

// hydrateSessionApplication hydrates the full Application actor based on the address provided
func (sh *sessionHydrator) hydrateSessionApplication() error {
	app, appIsFound := sh.k.appKeeper.GetApplication(*sh.ctx, sh.sessionHeader.ApplicationAddress)
	if !appIsFound {
		return sdkerrors.Wrapf(types.ErrHydratingSession, "failed to find session application")
	}
	sh.session.Application = &app
	return nil
}

// hydrateSessionSuppliers finds the suppliers that are staked at the session height and populates the session with them
func (sh *sessionHydrator) hydrateSessionSuppliers() error {
	// TECHDEBT(@Olshansk): Need to retrieve the suppliers at SessionStartBlockHeight, NOT THE CURRENT ONE
	// retrieve the suppliers at the current block height
	suppliers := sh.k.supplierKeeper.GetAllSupplier(*sh.ctx)

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
		sh.logger.Info("number of available suppliers (%d) is less than the number of suppliers per session (%d)", len(candidateSuppliers), NumSupplierPerSession)
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

func concat(b ...[]byte) (result []byte) {
	for _, bz := range b {
		result = append(result, bz...)
	}
	return result
}

func shas3Hash(bz []byte) []byte {
	hasher := crypto.SHA3_256.New()
	hasher.Write(bz)
	return hasher.Sum(nil)
}
