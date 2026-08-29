package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/session/types"
)

// Block-scoped memoization of hydrated sessions.
//
// Why: hydrating a session walks every supplier service-config update for the
// service at the requested height, unmarshals each record, derives a sha3 weight
// per candidate and sorts them (hydrateSessionSuppliers). A mainnet claim block
// carries ~3,300 MsgCreateClaim for only ~80 distinct sessions - one claim per
// supplier in the session - so the identical session was hydrated ~43 times per
// block, accounting for ~90% of the block's apply time (~37 s of a 60 s cadence)
// and the precommits validators drop at every session boundary.
//
// Where: the module's TRANSIENT store. It is
//   - part of the multistore branch each tx executes against: a memo written by
//     tx N is visible to tx N+1 of the same block, and discarded with the tx's
//     writes if tx N fails;
//   - wiped on Commit, so nothing survives the block;
//   - identical on every node, because it is populated only while executing
//     FinalizeBlock (memoEnabled), i.e. the same txs in the same order. CheckTx,
//     simulation, proposal handling and gRPC queries never read or write it, so
//     no node-local traffic can influence it (the failure mode of the previous
//     keeper-level in-memory cache).
//
// Why a memo hit equals a fresh hydration within the block: every input that
// decides the supplier set is a pure function of committed state at the
// requested height - shared/session params at height, the block hash at the
// session start height, and the supplier service-config-update index bounded by
// activation <= height. In-block stake/unstake writes stamp activation and
// deactivation heights at the NEXT session start, which is > the current height,
// so they cannot change a session requested at or before the current height.
// The live reads left in hydration (application record, dehydrated supplier
// record) decorate objects that the FinalizeBlock callers (x/proof claim and
// proof validation) do not use for validation - they check session ID and
// supplier membership, both derived from the historical inputs above.
//
// Gas: memo reads and writes are NOT gas-metered (infinite gas meter). Simulation
// runs with the memo disabled, so a simulated claim/proof tx pays for a cold
// hydration; a metered memo write would make the first (miss) tx per session
// per block cost MORE than its simulation, fail out of gas under `--gas auto`,
// and - since a failed tx's writes are discarded - leave the session
// un-memoized for every later tx in the block. Unmetered, a miss costs exactly
// what a fresh hydration costs today and a hit costs less, so a simulation is
// always an upper bound. The unmetered work is bounded by the metered
// hydration that precedes every write, so nothing is free to amplify.
//
// Consensus impact: a memo hit skips gas-metered store reads, so gas_used of
// claim/proof txs changes. That alters LastResultsHash (block header), not the
// AppHash - fees are charged on gas_limit, so state is identical with and
// without the memo. Activating this is therefore a coordinated upgrade.

// memoEnabled reports whether the block-scoped session memo is active for ctx:
// only while executing FinalizeBlock and only when a transient store service
// was wired into the keeper.
func (k Keeper) memoEnabled(ctx context.Context) bool {
	return k.transientStoreService != nil &&
		sdk.UnwrapSDKContext(ctx).ExecMode() == sdk.ExecModeFinalize
}

// getSessionMemoStore returns the prefixed transient store holding memoized
// sessions, opened with an infinite gas meter so memo traffic is never charged
// to the tx (see the gas note at the top of this file).
func (k Keeper) getSessionMemoStore(ctx context.Context) prefix.Store {
	unmeteredCtx := sdk.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewInfiniteGasMeter())
	storeAdapter := runtime.KVStoreAdapter(k.transientStoreService.OpenTransientStore(unmeteredCtx))
	return prefix.NewStore(storeAdapter, types.SessionMemoKeyPrefix)
}

// getMemoizedSession returns the session memoized under key within the current
// block, if the memo is enabled and populated. It returns a fresh unmarshaled
// copy on every hit so callers can never mutate the memoized value.
func (k Keeper) getMemoizedSession(ctx context.Context, key []byte) (*types.Session, bool) {
	if !k.memoEnabled(ctx) {
		return nil, false
	}

	sessionBz := k.getSessionMemoStore(ctx).Get(key)
	if sessionBz == nil {
		return nil, false
	}

	var session types.Session
	if err := k.cdc.Unmarshal(sessionBz, &session); err != nil {
		// A corrupt memo entry must never change the outcome: fall back to hydration.
		k.Logger().With("method", "getMemoizedSession").Error(err.Error())
		return nil, false
	}
	return &session, true
}

// memoizeSession stores session under key for the remainder of the current
// block when the memo is enabled. It is a no-op otherwise.
func (k Keeper) memoizeSession(ctx context.Context, key []byte, session *types.Session) {
	if !k.memoEnabled(ctx) {
		return
	}

	sessionBz, err := k.cdc.Marshal(session)
	if err != nil {
		k.Logger().With("method", "memoizeSession").Error(err.Error())
		return
	}
	k.getSessionMemoStore(ctx).Set(key, sessionBz)
}
