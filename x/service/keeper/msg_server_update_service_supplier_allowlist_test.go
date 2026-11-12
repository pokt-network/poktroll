package keeper_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/service/keeper"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	allowlistTestServiceId = "svc-allowlist"
	// allowlistTestHeight is the block height the fixture's context starts at.
	allowlistTestHeight int64 = 100
)

// allowlistFixture is a service keeper holding one service, owned by owner.
type allowlistFixture struct {
	k     keeper.Keeper
	srv   types.MsgServer
	ctx   sdk.Context
	owner string
}

func newAllowlistFixture(t *testing.T) allowlistFixture {
	t.Helper()

	k, ctx := keepertest.ServiceKeeper(t)
	owner := sample.AccAddressBech32()
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(allowlistTestHeight)
	k.SetService(sdkCtx, sharedtypes.Service{
		Id:                   allowlistTestServiceId,
		Name:                 "allowlist test service",
		ComputeUnitsPerRelay: 1,
		OwnerAddress:         owner,
	})

	return allowlistFixture{k: k, srv: keeper.NewMsgServerImpl(k), ctx: sdkCtx, owner: owner}
}

// update sends the owner's allowlist update and returns the allowlist events it emitted.
func (f allowlistFixture) update(t *testing.T, add, remove []string) []*types.EventServiceSupplierAllowlistUpdated {
	t.Helper()

	ctx := f.ctx.WithEventManager(sdk.NewEventManager())
	_, err := f.srv.UpdateServiceSupplierAllowlist(ctx, &types.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: f.owner,
		ServiceId:    allowlistTestServiceId,
		Add:          add,
		Remove:       remove,
	})
	require.NoError(t, err)

	return testevents.FilterEvents[*types.EventServiceSupplierAllowlistUpdated](t, ctx.EventManager().Events())
}

func TestUpdateServiceSupplierAllowlist_Authorization(t *testing.T) {
	f := newAllowlistFixture(t)
	supplierAddr := sample.AccAddressBech32()

	// A non-owner is rejected and nothing is written.
	_, err := f.srv.UpdateServiceSupplierAllowlist(f.ctx, &types.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: sample.AccAddressBech32(),
		ServiceId:    allowlistTestServiceId,
		Add:          []string{supplierAddr},
	})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.ErrorContains(t, err, types.ErrServiceUnauthorized.Error())
	require.False(t, f.k.IsServicePermissioned(f.ctx, allowlistTestServiceId))
	require.Empty(t, f.k.GetPendingAllowlistSweeps(f.ctx))

	// An unknown service is rejected.
	_, err = f.srv.UpdateServiceSupplierAllowlist(f.ctx, &types.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: f.owner,
		ServiceId:    "unknown-svc",
		Add:          []string{supplierAddr},
	})
	require.Equal(t, codes.NotFound, status.Code(err))
	require.ErrorContains(t, err, types.ErrServiceNotFound.Error())
	require.False(t, f.k.IsServicePermissioned(f.ctx, "unknown-svc"))

	// A message failing ValidateBasic is rejected as an invalid argument.
	_, err = f.srv.UpdateServiceSupplierAllowlist(f.ctx, &types.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: f.owner,
		ServiceId:    allowlistTestServiceId,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, types.ErrServiceInvalidSupplierAllowlist.Error())
}

func TestUpdateServiceSupplierAllowlist_Changes(t *testing.T) {
	f := newAllowlistFixture(t)
	addrA, addrB, addrC := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

	// Adding A and B lists both, in message order, and permissions the service.
	events := f.update(t, []string{addrA, addrB}, nil)
	require.Len(t, events, 1)
	require.Equal(t, allowlistTestServiceId, events[0].ServiceId)
	require.Equal(t, f.owner, events[0].OwnerAddress)
	require.True(t, events[0].Permissioned)
	require.Equal(t, []string{addrA, addrB}, events[0].Added)
	require.Empty(t, events[0].Removed)
	require.True(t, f.k.HasSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrA))
	require.True(t, f.k.HasSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrB))

	// Re-adding A and removing the absent C change nothing: the event lists nothing.
	events = f.update(t, []string{addrA}, []string{addrC})
	require.Len(t, events, 1)
	require.True(t, events[0].Permissioned)
	require.Empty(t, events[0].Added)
	require.Empty(t, events[0].Removed)

	// Removing A and B empties the list and lifts the restriction.
	events = f.update(t, nil, []string{addrA, addrB})
	require.Len(t, events, 1)
	require.False(t, events[0].Permissioned)
	require.Empty(t, events[0].Added)
	require.Equal(t, []string{addrA, addrB}, events[0].Removed)
	require.False(t, f.k.IsServicePermissioned(f.ctx, allowlistTestServiceId))
}

func TestUpdateServiceSupplierAllowlist_SweepMarking(t *testing.T) {
	addrA, addrB, addrC := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

	started := &types.AllowlistSweep{Cursor: []byte{1}}
	startedRerun := &types.AllowlistSweep{Cursor: []byte{1}, Rerun: true}
	deferred := &types.AllowlistSweep{DeferredFromHeight: 7}
	deferredNow := &types.AllowlistSweep{DeferredFromHeight: allowlistTestHeight}
	immediate := &types.AllowlistSweep{}

	tests := []struct {
		desc string
		// listed entries and the pending sweep (nil: none) before the update.
		listed      []string
		pending     *types.AllowlistSweep
		add, remove []string
		// expected pending sweep after the update (nil: none).
		expected *types.AllowlistSweep
	}{
		{
			desc: "first add schedules a sweep deferred from the block height",
			add:  []string{addrA}, expected: deferredNow,
		},
		{
			desc: "first add replaces whatever stale sweep row is there",
			add:  []string{addrA}, pending: startedRerun, expected: deferredNow,
		},
		{
			desc:   "real removal leaving a non-empty list starts a sweep now",
			listed: []string{addrA, addrB}, remove: []string{addrA},
			expected: immediate,
		},
		{
			desc:   "real removal on a started sweep sets rerun and keeps the cursor",
			listed: []string{addrA, addrB}, pending: started, remove: []string{addrA},
			expected: startedRerun,
		},
		{
			desc:   "real removal on a deferred sweep changes nothing",
			listed: []string{addrA, addrB}, pending: deferred, remove: []string{addrA},
			expected: deferred,
		},
		{
			desc:   "removal of an absent entry changes nothing",
			listed: []string{addrA}, remove: []string{addrC},
		},
		{
			desc:   "removal of an absent entry does not set rerun",
			listed: []string{addrA}, pending: started, remove: []string{addrC},
			expected: started,
		},
		{
			desc:   "removal of an absent entry from an unpermissioned service changes nothing",
			remove: []string{addrC},
		},
		{
			desc:   "add alone does not schedule a sweep on a permissioned service",
			listed: []string{addrA}, add: []string{addrB},
		},
		{
			desc:   "add alone leaves a started sweep untouched",
			listed: []string{addrA}, pending: started, add: []string{addrB},
			expected: started,
		},
		{
			desc:   "add alone leaves a deferred sweep untouched",
			listed: []string{addrA}, pending: deferred, add: []string{addrB},
			expected: deferred,
		},
		{
			desc:   "removing the last entry clears a started sweep",
			listed: []string{addrA}, pending: startedRerun, remove: []string{addrA},
		},
		{
			desc:   "removing the last entry clears a deferred sweep",
			listed: []string{addrA}, pending: deferred, remove: []string{addrA},
		},
		{
			desc:   "removing the last entry without a pending sweep leaves none",
			listed: []string{addrA}, remove: []string{addrA},
		},
		{
			desc:   "replacing the only entry starts a sweep now",
			listed: []string{addrA}, add: []string{addrB}, remove: []string{addrA},
			expected: immediate,
		},
		{
			desc:   "replacing the only entry on a started sweep sets rerun",
			listed: []string{addrA}, pending: started, add: []string{addrB}, remove: []string{addrA},
			expected: startedRerun,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			f := newAllowlistFixture(t)
			for _, address := range test.listed {
				f.k.SetSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, address)
			}
			if test.pending != nil {
				pending := *test.pending
				pending.ServiceId = allowlistTestServiceId
				f.k.SetAllowlistSweep(f.ctx, pending)
			}

			f.update(t, test.add, test.remove)

			if test.expected == nil {
				requireSweeps(t, f.k, f.ctx)
				return
			}
			expected := *test.expected
			expected.ServiceId = allowlistTestServiceId
			requireSweeps(t, f.k, f.ctx, expected)
		})
	}
}

// Removals keep requesting sweeps of a started sweep, one per block, without
// ever restarting it: the cursor stays and rerun stays set.
func TestUpdateServiceSupplierAllowlist_RemovalSpam(t *testing.T) {
	f := newAllowlistFixture(t)
	const numRemovals = 10

	// One more entry than removals: the list never empties.
	addresses := make([]string, numRemovals+1)
	for i := range addresses {
		addresses[i] = sample.AccAddressBech32()
		f.k.SetSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addresses[i])
	}
	f.k.SetAllowlistSweep(f.ctx, types.AllowlistSweep{ServiceId: allowlistTestServiceId, Cursor: []byte{1}})

	for i := 0; i < numRemovals; i++ {
		f.ctx = f.ctx.WithBlockHeight(allowlistTestHeight + int64(i))
		f.update(t, nil, []string{addresses[i]})

		requireSweeps(t, f.k, f.ctx, types.AllowlistSweep{
			ServiceId: allowlistTestServiceId,
			Cursor:    []byte{1},
			Rerun:     true,
		})
	}
	require.True(t, f.k.IsServicePermissioned(f.ctx, allowlistTestServiceId))
}

func TestUpdateServiceSupplierAllowlist_ReplaceOnlyEntry(t *testing.T) {
	f := newAllowlistFixture(t)
	addrA, addrB := sample.AccAddressBech32(), sample.AccAddressBech32()
	f.k.SetSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrA)

	// The service stays permissioned, but A's suppliers must still leave.
	events := f.update(t, []string{addrB}, []string{addrA})
	require.Len(t, events, 1)
	require.True(t, events[0].Permissioned)
	require.Equal(t, []string{addrB}, events[0].Added)
	require.Equal(t, []string{addrA}, events[0].Removed)

	require.True(t, f.k.IsServicePermissioned(f.ctx, allowlistTestServiceId))
	require.False(t, f.k.HasSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrA))
	require.True(t, f.k.HasSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrB))

	requireSweeps(t, f.k, f.ctx, types.AllowlistSweep{ServiceId: allowlistTestServiceId})
}

func TestTransferService_KeepsSupplierAllowlist(t *testing.T) {
	f := newAllowlistFixture(t)
	addrA, addrB := sample.AccAddressBech32(), sample.AccAddressBech32()
	newOwner := sample.AccAddressBech32()

	f.update(t, []string{addrA}, nil)
	f.k.SetAllowlistSweep(f.ctx, types.AllowlistSweep{ServiceId: allowlistTestServiceId, Cursor: []byte{1}})

	_, err := f.srv.TransferService(f.ctx, types.NewMsgTransferService(f.owner, allowlistTestServiceId, newOwner))
	require.NoError(t, err)

	// The transfer leaves the allowlist and its pending sweep as they were.
	require.True(t, f.k.HasSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrA))
	require.True(t, f.k.IsServicePermissioned(f.ctx, allowlistTestServiceId))
	sweeps := f.k.GetPendingAllowlistSweeps(f.ctx)
	require.Len(t, sweeps, 1)
	require.Equal(t, allowlistTestServiceId, sweeps[0].ServiceId)
	require.Equal(t, []byte{1}, sweeps[0].Cursor)

	// The old owner lost control of the allowlist.
	_, err = f.srv.UpdateServiceSupplierAllowlist(f.ctx, &types.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: f.owner,
		ServiceId:    allowlistTestServiceId,
		Add:          []string{addrB},
	})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.ErrorContains(t, err, types.ErrServiceUnauthorized.Error())
	require.False(t, f.k.HasSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrB))

	// The new owner has it.
	_, err = f.srv.UpdateServiceSupplierAllowlist(f.ctx, &types.MsgUpdateServiceSupplierAllowlist{
		OwnerAddress: newOwner,
		ServiceId:    allowlistTestServiceId,
		Add:          []string{addrB},
	})
	require.NoError(t, err)
	require.True(t, f.k.HasSupplierAllowlistEntry(f.ctx, allowlistTestServiceId, addrB))
}

// The owner check and the entries are string-keyed, so only the canonical
// encoding of an address is ever accepted, on add and on remove, and a service
// whose owner was stored in another encoding regains control only once
// CanonicalizeServiceOwnerAddresses has rewritten it.
func TestUpdateServiceSupplierAllowlist_CanonicalOwnerAndEntries(t *testing.T) {
	k, ctx := keepertest.ServiceKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(allowlistTestHeight)
	srv := keeper.NewMsgServerImpl(k)
	owner, supplierOwner, operator := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()
	const svc = "svc-upper"

	k.SetService(sdkCtx, sharedtypes.Service{Id: svc, Name: "upper", ComputeUnitsPerRelay: 1, OwnerAddress: strings.ToUpper(owner)})

	update := func(signer string, add, remove []string) error {
		_, err := srv.UpdateServiceSupplierAllowlist(sdkCtx.WithEventManager(sdk.NewEventManager()), &types.MsgUpdateServiceSupplierAllowlist{
			OwnerAddress: signer, ServiceId: svc, Add: add, Remove: remove,
		})
		return err
	}

	// Before canonicalization the owner is locked out in both encodings.
	err := update(owner, []string{supplierOwner}, nil)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	err = update(strings.ToUpper(owner), []string{supplierOwner}, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.False(t, k.IsServicePermissioned(sdkCtx, svc))

	require.Equal(t, 1, k.CanonicalizeServiceOwnerAddresses(sdkCtx))
	require.NoError(t, update(owner, []string{supplierOwner}, nil))

	// Non-canonical entries are rejected by the handler on add AND remove.
	err = update(owner, []string{strings.ToUpper(operator)}, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, types.ErrServiceInvalidSupplierAllowlist.Error())
	err = update(owner, nil, []string{strings.ToUpper(supplierOwner)})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.True(t, k.HasSupplierAllowlistEntry(sdkCtx, svc, supplierOwner), "a non-canonical remove must not touch the canonical entry")
	require.False(t, k.HasSupplierAllowlistEntry(sdkCtx, svc, strings.ToUpper(operator)))

	// The lookup never matches another encoding of a listed address.
	require.True(t, k.IsSupplierAllowed(sdkCtx, svc, supplierOwner, operator))
	require.False(t, k.IsSupplierAllowed(sdkCtx, svc, strings.ToUpper(supplierOwner), operator))
}
