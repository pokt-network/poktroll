package keeper

import (
	"context"
	"fmt"
	"sync"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/session/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		accountKeeper     types.AccountKeeper
		bankKeeper        types.BankKeeper
		applicationKeeper types.ApplicationKeeper
		supplierKeeper    types.SupplierKeeper
		sharedKeeper      types.SharedKeeper

		// Session cache to reduce repeated iterator calls
		// Key format: "appAddr:serviceId:sessionNumber"
		sessionCache *sessionCache
	}
)

type sessionCache struct {
	cache   map[string]*types.Session
	mu      sync.RWMutex
	maxSize int
}

func newSessionCache(maxSize int) *sessionCache {
	return &sessionCache{
		cache:   make(map[string]*types.Session, maxSize),
		maxSize: maxSize,
	}
}

// sessionCacheKey generates a cache key from session parameters
// Format: "appAddr:serviceId:sessionNumber"
func sessionCacheKey(appAddr, serviceId string, sessionNumber int64) string {
	return fmt.Sprintf("%s:%s:%d", appAddr, serviceId, sessionNumber)
}

func (sc *sessionCache) get(key string) (*types.Session, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	session, ok := sc.cache[key]
	return session, ok
}

func (sc *sessionCache) set(key string, session *types.Session) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Simple FIFO eviction if cache is full
	if len(sc.cache) >= sc.maxSize {
		// Delete a random entry (Go map iteration is randomized)
		for k := range sc.cache {
			delete(sc.cache, k)
			break
		}
	}

	sc.cache[key] = session
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	applicationKeeper types.ApplicationKeeper,
	supplierKeeper types.SupplierKeeper,
	sharedKeeper types.SharedKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	// Initialize session cache with reasonable default size
	// This caches up to 10000 sessions to reduce iterator overhead
	const defaultSessionCacheSize = 10000

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		accountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		applicationKeeper: applicationKeeper,
		supplierKeeper:    supplierKeeper,
		sharedKeeper:      sharedKeeper,
		sessionCache:      newSessionCache(defaultSessionCacheSize),
	}
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// StoreBlockHash is called at the end of every block.
// It fetches the block hash from the committed block ans saves its hash
// in the store.
func (k Keeper) StoreBlockHash(goCtx context.Context) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ctx.HeaderHash() is the hash of the block being validated.
	hash := ctx.HeaderHash()

	// ctx.BlocHeight() is the height of the block being validated.
	height := ctx.BlockHeight()

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(goCtx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.BlockHashKeyPrefix))
	store.Set(types.BlockHashKey(height), hash)
}
