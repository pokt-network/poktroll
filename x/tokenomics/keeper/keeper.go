package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TokenomicsKeeperI is the interface contract that x/tokenomics's keeper implements.
type TokenomicsKeeperI interface {
	// GetAuthority returns the x/tokenomics module's authority.
	GetAuthority() string
}

// TODO_TECHDEBT(#240): See `x/auth/keeper.keeper.go` in the Cosmos SDK on how
// we should refactor all out keepers. This one has started following a subset
// of those patterns as they related to parameters.
type TokenomicsKeeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	memKey     storetypes.StoreKey
	paramstore paramtypes.Subspace

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewTokenomicsKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	authority string,

) *TokenomicsKeeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	return &TokenomicsKeeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,

		authority: authority,
	}
}

func (k TokenomicsKeeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the x/tokenomics module's authority.
func (k TokenomicsKeeper) GetAuthority() string {
	return k.authority
}
