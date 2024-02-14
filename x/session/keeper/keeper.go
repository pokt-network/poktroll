package keeper

import (
	"context"
	"fmt"

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
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	applicationKeeper types.ApplicationKeeper,
	supplierKeeper types.SupplierKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		accountKeeper:     accountKeeper,
		bankKeeper:        bankKeeper,
		applicationKeeper: applicationKeeper,
		supplierKeeper:    supplierKeeper,
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

// BeginBlocker is called at the beginning of every block.
// It fetches the block hash from the committed block ans saves its hash
// in the store.
func (k Keeper) BeginBlocker(goCtx context.Context) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// ctx.BlockHeader().AppHash is the hash of the last block committed
	hash := ctx.BlockHeader().AppHash
	// ctx.BlockHeader().Height is the height of the last committed block.
	height := ctx.BlockHeader().Height

	storeAdapter := runtime.KVStoreAdapter(k.storeService.OpenKVStore(goCtx))
	store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SessionKeyPrefix))
	store.Set(types.SessionKey(height), hash)
}
