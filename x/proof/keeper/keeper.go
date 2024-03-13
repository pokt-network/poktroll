package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/crypto"
	pubkeyclient "github.com/pokt-network/poktroll/pkg/crypto/pubkey_client"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/x/proof/types"
)

type (
	Keeper struct {
		cdc          codec.BinaryCodec
		storeService store.KVStoreService
		logger       log.Logger

		// the address capable of executing a MsgUpdateParams message. Typically, this
		// should be the x/gov module account.
		authority string

		sessionKeeper     types.SessionKeeper
		applicationKeeper types.ApplicationKeeper
		accountKeeper     types.AccountKeeper

		ringClient   crypto.RingClient
		pubKeyClient crypto.PubKeyClient
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	sessionKeeper types.SessionKeeper,
	applicationKeeper types.ApplicationKeeper,
	accountKeeper types.AccountKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	applicationQuerier := types.NewAppKeeperQueryClient(applicationKeeper)
	accountQuerier := types.NewAccountKeeperQueryClient(accountKeeper)
	// TODO_TECHDEBT: Use cosmos-sdk based polylog implementation once available. Also remove polyzero import.
	polylogger := polylog.Ctx(context.TODO())

	ringClientDeps := depinject.Supply(
		polylogger,
		applicationQuerier,
		accountQuerier,
	)

	ringClient, err := rings.NewRingClient(ringClientDeps)
	if err != nil {
		panic(err)
	}

	pubKeyClient, err := pubkeyclient.NewPubKeyClient(depinject.Supply(accountQuerier))
	if err != nil {
		panic(err)
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		sessionKeeper:     sessionKeeper,
		applicationKeeper: applicationKeeper,
		accountKeeper:     accountKeeper,
		ringClient:        ringClient,
		pubKeyClient:      pubKeyClient,
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
