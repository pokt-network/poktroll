package keeper

import (
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	logger       log.Logger

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string

	// keepers
	bankKeeper        types.BankKeeper
	accountKeeper     types.AccountKeeper
	applicationKeeper types.ApplicationKeeper
	proofKeeper       types.ProofKeeper
	sharedKeeper      types.SharedKeeper
	sessionKeeper     types.SessionKeeper
	supplierKeeper    types.SupplierKeeper

	sharedQuerier client.SharedQueryClient
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	applicationKeeper types.ApplicationKeeper,
	proofKeeper types.ProofKeeper,
	sharedKeeper types.SharedKeeper,
	sessionKeeper types.SessionKeeper,
	supplierKeeper types.SupplierKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sharedQuerier := prooftypes.NewSharedKeeperQueryClient(sharedKeeper, sessionKeeper)

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		bankKeeper:        bankKeeper,
		accountKeeper:     accountKeeper,
		applicationKeeper: applicationKeeper,
		proofKeeper:       proofKeeper,
		sharedKeeper:      sharedKeeper,
		sessionKeeper:     sessionKeeper,
		supplierKeeper:    supplierKeeper,

		sharedQuerier: sharedQuerier,
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
