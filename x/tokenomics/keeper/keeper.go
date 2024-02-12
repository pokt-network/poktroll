package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// Keeper is the structure that implements the `KeeperI` interface.

// TODO_TECHDEBT(#240): See `x/nft/keeper.keeper.go` in the Cosmos SDK on how
// we should refactor all our keepers. This keeper has started following a small
// subset of those patterns.
type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		bankKeeper        types.BankKeeper
		accountKeeper     types.AccountKeeper
		applicationKeeper types.ApplicationKeeper
		supplierKeeper    types.SupplierKeeper
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
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

		bankKeeper:        bankKeeper,
		accountKeeper:     accountKeeper,
		applicationKeeper: applicationKeeper,
		supplierKeeper:    supplierKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger() log.Logger {
	return k.logger.With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the x/tokenomics module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}
