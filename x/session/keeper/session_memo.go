package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Block-scoped memoization of session supplier selection.
//
// Why: selecting a session's suppliers walks every supplier service-config
// update for the service at the requested height, unmarshals each record,
// derives a sha3 weight per candidate and sorts them (hydrateSessionSuppliers).
// A mainnet claim block carries ~3,300 MsgCreateClaim for only ~80 distinct
// sessions - one claim per supplier in the session - so the identical selection
// was recomputed ~43 times per block, accounting for ~90% of the block's apply
// time (~37 s of a 60 s cadence) and the precommits validators drop at every
// session boundary.
//
// What: ONLY the selected service-config updates (top NumSuppliersPerSession by
// weight, or every candidate when there are fewer). Everything else in the
// session - params at height, block hash, session ID, the application record and
// the supplier records - is still read on every call, hit or miss, so a hit
// returns exactly what a fresh hydration returns, including records written
// earlier in the same block.
//
// Where: the module's TRANSIENT store. It is
//   - part of the multistore branch each tx executes against: a memo written by
//     tx N is visible to tx N+1 of the same block, and discarded with the tx's
//     writes if tx N fails;
//   - wiped on Commit, and keyed by the current height, so nothing is served
//     outside the block that wrote it;
//   - identical on every node, because it is populated only while executing
//     FinalizeBlock (sessionMemoStore), i.e. the same txs in the same order.
//     CheckTx, simulation, proposal handling and gRPC queries never read or
//     write it, so no node-local traffic can influence it (the failure mode of
//     the previous keeper-level in-memory cache).
//
// Why a memoized selection equals a fresh one: the key (types.SessionMemoKey)
// holds every input of the selection - the requested height, the session ID
// bytes (block hash at session start, service ID, app address, session start
// height from shared params at height) and NumSuppliersPerSession at height. A
// param change inside the block, including one that reaches a past height through
// the live-params fallback of GetParamsAtHeight, changes the key and misses. The
// remaining input is the service-config-update index bounded by
// activation <= height < deactivation. Every tx that writes it (stake, restake,
// unstake) stamps activation or deactivation at the NEXT session start, which is
// above the current height and so above any height a session is requested for.
// EndBlocker writes to the index (settlement force-unstake, unbonding, pruning)
// run after every GetSession caller of the block (claim and proof txs).
//
// Gas: memo reads and writes are NOT gas-metered (infinite gas meter). Simulation
// runs with the memo disabled, so a simulated claim/proof tx pays for a cold
// hydration; a metered memo write would make the first (miss) tx per session
// per block cost MORE than its simulation, fail out of gas under `--gas auto`,
// and - since a failed tx's writes are discarded - leave the session
// un-memoized for every later tx in the block. Unmetered, a miss costs exactly
// what a fresh hydration costs and a hit costs less (it skips only the
// service-config walk), so a simulation is always an upper bound. The unmetered
// work is bounded by the metered walk that precedes every write, so nothing is
// free to amplify.
//
// Consensus impact: a memo hit skips gas-metered store reads, so gas_used of
// claim/proof txs changes. That alters LastResultsHash (block header), not the
// AppHash - fees are charged on gas_limit, so state is identical with and
// without the memo. Activating this is therefore a coordinated upgrade.

// sessionMemoStore returns the prefixed transient store holding memoized
// supplier selections, or nil when the memo is off: outside FinalizeBlock, or
// when no transient store service was wired into the keeper. It is opened with
// an infinite gas meter so memo traffic is never charged to the tx (see the gas
// note at the top of this file).
func (k Keeper) sessionMemoStore(ctx context.Context) storetypes.KVStore {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if k.transientStoreService == nil || sdkCtx.ExecMode() != sdk.ExecModeFinalize {
		return nil
	}

	unmeteredCtx := sdkCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	storeAdapter := runtime.KVStoreAdapter(k.transientStoreService.OpenTransientStore(unmeteredCtx))
	return prefix.NewStore(storeAdapter, types.SessionMemoKeyPrefix)
}

// getMemoizedSupplierConfigs returns the supplier selection memoized under key,
// as freshly unmarshaled values so callers can never mutate the memoized one.
func (k Keeper) getMemoizedSupplierConfigs(memoStore storetypes.KVStore, key []byte) ([]*sharedtypes.ServiceConfigUpdate, bool) {
	selectionBz := memoStore.Get(key)
	if selectionBz == nil {
		return nil, false
	}

	var selection sharedtypes.Supplier
	if err := k.cdc.Unmarshal(selectionBz, &selection); err != nil {
		// A corrupt memo entry must never change the outcome: fall back to the walk.
		k.Logger().With("method", "getMemoizedSupplierConfigs").Error(err.Error())
		return nil, false
	}
	return selection.ServiceConfigHistory, true
}

// memoizeSupplierConfigs stores a supplier selection under key for the
// remainder of the current block.
func (k Keeper) memoizeSupplierConfigs(memoStore storetypes.KVStore, key []byte, selected []*sharedtypes.ServiceConfigUpdate) {
	// Supplier is only a proto container here: its service_config_history field
	// is exactly a list of ServiceConfigUpdate, so no new proto type is needed
	// for data that never leaves the block.
	selectionBz, err := k.cdc.Marshal(&sharedtypes.Supplier{ServiceConfigHistory: selected})
	if err != nil {
		k.Logger().With("method", "memoizeSupplierConfigs").Error(err.Error())
		return
	}
	memoStore.Set(key, selectionBz)
}
