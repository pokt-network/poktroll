package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto"
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

		bankKeeper        types.BankKeeper
		sessionKeeper     types.SessionKeeper
		applicationKeeper types.ApplicationKeeper
		accountKeeper     types.AccountKeeper
		sharedKeeper      types.SharedKeeper

		ringClient     crypto.RingClient
		accountQuerier client.AccountQueryClient
		sharedQuerier  client.SharedQueryClient
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	logger log.Logger,
	authority string,

	bankKeeper types.BankKeeper,
	sessionKeeper types.SessionKeeper,
	applicationKeeper types.ApplicationKeeper,
	accountKeeper types.AccountKeeper,
	sharedKeeper types.SharedKeeper,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	// TODO_TECHDEBT: Use cosmos-sdk based polylog implementation once available. Also remove polyzero import.
	polylogger := polylog.Ctx(context.Background())
	applicationQuerier := types.NewAppKeeperQueryClient(applicationKeeper)
	accountQuerier := types.NewAccountKeeperQueryClient(accountKeeper)
	sharedQuerier := types.NewSharedKeeperQueryClient(sharedKeeper, sessionKeeper)

	// RingKeeperClient holds the logic of verifying RelayRequests ring signatures
	// for both on-chain and off-chain actors.
	//
	// ApplicationQueriers & AccountQuerier are compatible with the environment
	// they're used in and may or may not make an actual network request.
	//
	// When used in an on-chain context, the ProofKeeper supplies AppKeeperQueryClient
	// and AccountKeeperQueryClient that are thin wrappers around the Application and
	// Account keepers respectively to satisfy the RingClient needs.
	//
	// TODO_MAINNET(@red-0ne): Make ring signature verification a stateless
	// function and get rid of the RingClient and its dependencies by moving
	// application ring retrieval to the application keeper, and making it
	// retrievable using the application query client for off-chain actors. Signature
	// verification code will still be shared across off/on chain environments.
	ringKeeperClientDeps := depinject.Supply(polylogger, applicationQuerier, accountQuerier, sharedQuerier)
	ringKeeperClient, err := rings.NewRingClient(ringKeeperClientDeps)
	if err != nil {
		panic(err)
	}

	return Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,
		logger:       logger,

		bankKeeper:        bankKeeper,
		sessionKeeper:     sessionKeeper,
		applicationKeeper: applicationKeeper,
		accountKeeper:     accountKeeper,
		sharedKeeper:      sharedKeeper,

		ringClient:     ringKeeperClient,
		accountQuerier: accountQuerier,
		sharedQuerier:  sharedQuerier,
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
