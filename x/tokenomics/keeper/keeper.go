package keeper

import (
	"context"
	"fmt"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	"github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TokenomicsKeeperI is the interface contract that x/tokenomics's keeper implements.
type TokenomicsKeeperI interface {
	// GetAuthority returns the x/tokenomics module's authority.
	GetAuthority() string

	// SettleSessionAccounting is responsible for all of the post-session accounting
	// necessary to burn, mint or transfer tokens depending on the amount of work
	// done. The amount of "work done" complete is dictated by `sum` of `root`.
	//
	// ASSUMPTION: It is assumed the caller of this function validated the claim
	// against a proof BEFORE calling this function.

	// TODO_BLOCKER(@Olshansk): Is there a way to limit who can call this function?
	SettleSessionAccounting(goCtx context.Context, claim suppliertypes.Claim)
}

// TokenomicsKeeper is the structure that implements the `TokenomicsKeeperI` interface.
//
// TODO_TECHDEBT(#240): See `x/auth/keeper.keeper.go` in the Cosmos SDK on how
// we should refactor all our keepers. This keeper has started following a small
// subset of those patterns.
type TokenomicsKeeper struct {
	cdc        codec.BinaryCodec
	storeKey   storetypes.StoreKey
	memKey     storetypes.StoreKey
	paramstore paramtypes.Subspace

	// keeper dependencies
	bankKeeper types.BankKeeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewTokenomicsKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	// keeper dependencies
	bankKeeper types.BankKeeper,

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

		bankKeeper: bankKeeper,

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
