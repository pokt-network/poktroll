package keeper_test

import (
	"context"
	"strings"
	"testing"

	"cosmossdk.io/store/prefix"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// stakeCanonicalSupplier stakes a supplier with a service through the msg server.
func stakeCanonicalSupplier(t *testing.T, k keepertest.SupplierModuleKeepers, ctx context.Context, owner, operator string) {
	t.Helper()
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	stakeMsg, _ := newSupplierStakeMsg(owner, operator, suppliertypes.DefaultMinStake.Amount.Int64(), serviceID)
	_, err := srv.StakeSupplier(ctx, setStakeMsgSigner(stakeMsg, operator))
	require.NoError(t, err)
}

// setNonCanonicalSupplier writes a service-less supplier stored under the uppercase
// encoding of operator, as MsgStakeSupplier accepted before canonical addresses
// were required.
func setNonCanonicalSupplier(k keepertest.SupplierModuleKeepers, ctx context.Context, owner, operator string) sharedtypes.Supplier {
	stake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, suppliertypes.DefaultMinStake.Amount.Int64())
	supplier := sharedtypes.Supplier{
		OwnerAddress:    owner,
		OperatorAddress: strings.ToUpper(operator),
		Stake:           &stake,
	}
	k.SetAndIndexDehydratedSupplier(ctx, supplier)
	return supplier
}

func unstakeSupplier(t *testing.T, k keepertest.SupplierModuleKeepers, ctx context.Context, owner, operator string) {
	t.Helper()
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	_, err := srv.UnstakeSupplier(ctx, &suppliertypes.MsgUnstakeSupplier{Signer: owner, OperatorAddress: operator})
	require.NoError(t, err)
}

// runUnbondingEndBlockers runs the unbonding EndBlocker on a cache-wrapped context
// (as FinalizeBlock does) at numSessions consecutive session ends from startHeight.
func runUnbondingEndBlockers(t *testing.T, k keepertest.SupplierModuleKeepers, ctx context.Context, startHeight int64, numSessions int) context.Context {
	t.Helper()
	sharedParams := k.SharedKeeper.GetParams(ctx)
	for i := 0; i < numSessions; i++ {
		height := startHeight + int64(i)*int64(sharedParams.NumBlocksPerSession)
		blockCtx, write := cosmostypes.UnwrapSDKContext(keepertest.SetBlockHeight(ctx, height)).CacheContext()
		_, err := k.EndBlockerUnbondSuppliers(blockCtx)
		require.NoError(t, err)
		write()
		ctx = keepertest.SetBlockHeight(ctx, height)
	}
	return ctx
}

func TestMsgStakeSupplier_RejectsNonCanonicalAddresses(t *testing.T) {
	addr := sample.AccAddressBech32()
	upper := strings.ToUpper(addr)

	stakeMsg, _ := newSupplierStakeMsg(addr, addr, suppliertypes.DefaultMinStake.Amount.Int64(), serviceID)
	require.NoError(t, setStakeMsgSigner(stakeMsg, addr).ValidateBasic())

	for name, mutate := range map[string]func(*suppliertypes.MsgStakeSupplier){
		"owner":    func(m *suppliertypes.MsgStakeSupplier) { m.OwnerAddress = upper },
		"operator": func(m *suppliertypes.MsgStakeSupplier) { m.OperatorAddress = upper; m.Signer = upper },
		"signer":   func(m *suppliertypes.MsgStakeSupplier) { m.Signer = upper },
	} {
		t.Run(name, func(t *testing.T) {
			msg, _ := newSupplierStakeMsg(addr, addr, suppliertypes.DefaultMinStake.Amount.Int64(), serviceID)
			setStakeMsgSigner(msg, addr)
			mutate(msg)
			require.ErrorIs(t, msg.ValidateBasic(), suppliertypes.ErrSupplierInvalidAddress)
		})
	}

	// Without services any signer is accepted, but a non-canonical operator still is not.
	serviceless := &suppliertypes.MsgStakeSupplier{Signer: addr, OwnerAddress: addr, OperatorAddress: upper, Stake: stakeMsg.Stake}
	require.ErrorIs(t, serviceless.ValidateBasic(), suppliertypes.ErrSupplierInvalidAddress)

	for _, msg := range []*suppliertypes.MsgUnstakeSupplier{
		{Signer: upper, OperatorAddress: addr},
		{Signer: addr, OperatorAddress: upper},
	} {
		require.ErrorIs(t, msg.ValidateBasic(), suppliertypes.ErrSupplierInvalidAddress)
	}
	require.NoError(t, (&suppliertypes.MsgUnstakeSupplier{Signer: addr, OperatorAddress: addr}).ValidateBasic())
}

// A non-canonical supplier that finishes unbonding is refunded once and removed,
// and the supplier stored under the canonical encoding of the same account is untouched.
func TestEndBlockerUnbondSuppliers_NonCanonicalOperator_RefundedOnceAndRemoved(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sharedParams := k.SharedKeeper.GetParams(ctx)

	owner, operator := sample.AccAddressBech32(), sample.AccAddressBech32()
	stakeCanonicalSupplier(t, k, ctx, owner, operator)

	aliasOwner := sample.AccAddressBech32()
	alias := setNonCanonicalSupplier(k, ctx, aliasOwner, operator)
	k.BeginSupplierUnbonding(ctx, &alias, &sharedParams, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY)

	endHeight := k.GetSupplierUnbondingEndHeight(ctx, &alias)
	ctx = runUnbondingEndBlockers(t, k, ctx, endHeight, 3)

	require.Equal(t, alias.Stake.Amount.Int64(), k.SupplierBalanceMap[aliasOwner], "refunded exactly once")
	_, found := k.GetDehydratedSupplier(ctx, alias.OperatorAddress)
	require.False(t, found)
	iterator := k.GetAllUnstakingSuppliersIterator(ctx)
	require.False(t, iterator.Valid(), "unstaking index must be empty")
	iterator.Close()

	supplier, found := k.GetSupplier(ctx, operator)
	require.True(t, found)
	require.Equal(t, owner, supplier.OwnerAddress)
	require.False(t, supplier.IsUnbonding())
	require.Len(t, supplier.ServiceConfigHistory, 1)
}

// A non-canonical supplier and the canonical supplier of the same account finish
// unbonding in the same block: both are refunded once and removed, without error.
func TestEndBlockerUnbondSuppliers_NonCanonicalAndCanonicalSameBlock(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sharedParams := k.SharedKeeper.GetParams(ctx)

	owner, operator := sample.AccAddressBech32(), sample.AccAddressBech32()
	stakeCanonicalSupplier(t, k, ctx, owner, operator)
	unstakeSupplier(t, k, ctx, owner, operator)
	ownerBalanceBeforeRefund := k.SupplierBalanceMap[owner]

	aliasOwner := sample.AccAddressBech32()
	alias := setNonCanonicalSupplier(k, ctx, aliasOwner, operator)
	k.BeginSupplierUnbonding(ctx, &alias, &sharedParams, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY)

	canonical, found := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, found)
	endHeight := k.GetSupplierUnbondingEndHeight(ctx, &alias)
	require.Equal(t, endHeight, k.GetSupplierUnbondingEndHeight(ctx, &canonical))

	ctx = runUnbondingEndBlockers(t, k, ctx, endHeight, 2)

	require.Equal(t, alias.Stake.Amount.Int64(), k.SupplierBalanceMap[aliasOwner])
	require.Equal(t, canonical.Stake.Amount.Int64(), k.SupplierBalanceMap[owner]-ownerBalanceBeforeRefund)
	for _, address := range []string{alias.OperatorAddress, operator} {
		_, found := k.GetDehydratedSupplier(ctx, address)
		require.False(t, found, address)
	}
}

// An unstaking index entry without a supplier record is dropped instead of
// failing the EndBlocker.
func TestEndBlockerUnbondSuppliers_DanglingIndexEntryIsDropped(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)

	owner, operator := sample.AccAddressBech32(), sample.AccAddressBech32()
	stakeCanonicalSupplier(t, k, ctx, owner, operator)
	unstakeSupplier(t, k, ctx, owner, operator)
	supplier, found := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, found)
	endHeight := k.GetSupplierUnbondingEndHeight(ctx, &supplier)

	supplierStore := prefix.NewStore(cosmostypes.UnwrapSDKContext(ctx).KVStore(k.StoreKey), suppliertypes.KeyPrefix(suppliertypes.SupplierOperatorKeyPrefix))
	supplierStore.Delete(suppliertypes.SupplierOperatorKey(operator))

	ctx = runUnbondingEndBlockers(t, k, ctx, endHeight, 1)
	iterator := k.GetAllUnstakingSuppliersIterator(ctx)
	require.False(t, iterator.Valid(), "dangling index entry must be removed")
	iterator.Close()
}

func TestMigrateNonCanonicalSupplierAddresses(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sharedParams := k.SharedKeeper.GetParams(ctx)

	// Canonical supplier of the same account as the staked alias: must stay untouched.
	owner, operator := sample.AccAddressBech32(), sample.AccAddressBech32()
	stakeCanonicalSupplier(t, k, ctx, owner, operator)

	// Canonical supplier whose owner was stored non-canonical before the upgrade.
	upperOwnerOwner, upperOwnerOperator := sample.AccAddressBech32(), sample.AccAddressBech32()
	stakeCanonicalSupplier(t, k, ctx, upperOwnerOwner, upperOwnerOperator)
	upperOwnerSupplier, found := k.GetDehydratedSupplier(ctx, upperOwnerOperator)
	require.True(t, found)
	upperOwnerSupplier.OwnerAddress = strings.ToUpper(upperOwnerOwner)
	k.SetDehydratedSupplier(ctx, upperOwnerSupplier)

	// Aliases: unbonding already ended (refunded before the upgrade), unbonding
	// pending, and staked.
	var aliases []sharedtypes.Supplier
	unbonded := setNonCanonicalSupplier(k, ctx, sample.AccAddressBech32(), sample.AccAddressBech32())
	k.BeginSupplierUnbonding(ctx, &unbonded, &sharedParams, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY)
	upgradeHeight := k.GetSupplierUnbondingEndHeight(ctx, &unbonded) + 3
	ctx = keepertest.SetBlockHeight(ctx, upgradeHeight)
	pending := setNonCanonicalSupplier(k, ctx, sample.AccAddressBech32(), sample.AccAddressBech32())
	k.BeginSupplierUnbonding(ctx, &pending, &sharedParams, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY)
	staked := setNonCanonicalSupplier(k, ctx, sample.AccAddressBech32(), operator)
	aliases = append(aliases, unbonded, pending, staked)

	ctx, _ = testevents.ResetEventManager(ctx)
	numRemoved, numOwnersCanonicalized := k.MigrateNonCanonicalSupplierAddresses(ctx)
	require.Equal(t, 3, numRemoved)
	require.Equal(t, 1, numOwnersCanonicalized)

	eventCounts := make(map[string]int)
	for _, event := range cosmostypes.UnwrapSDKContext(ctx).EventManager().Events() {
		eventCounts[event.Type]++
	}
	require.Equal(t, 3, eventCounts[proto.MessageName(&suppliertypes.EventSupplierStakeStuckInModulePool{})])
	require.Equal(t, 3, eventCounts[proto.MessageName(&suppliertypes.EventSupplierUnbondingEnd{})])

	// Every alias is gone, with no index entry left, and is never refunded.
	ctx = runUnbondingEndBlockers(t, k, ctx, sharedtypes.GetSessionEndHeight(&sharedParams, upgradeHeight), 3)
	for _, alias := range aliases {
		_, isFound := k.GetDehydratedSupplier(ctx, alias.OperatorAddress)
		require.False(t, isFound, alias.OperatorAddress)
		require.Zero(t, k.SupplierBalanceMap[alias.OwnerAddress], "alias %s must not be refunded", alias.OperatorAddress)
	}
	iterator := k.GetAllUnstakingSuppliersIterator(ctx)
	require.False(t, iterator.Valid(), "unstaking index must be empty")
	iterator.Close()

	canonical, found := k.GetSupplier(ctx, operator)
	require.True(t, found)
	require.Equal(t, owner, canonical.OwnerAddress)
	require.False(t, canonical.IsUnbonding())
	require.Len(t, canonical.ServiceConfigHistory, 1)

	// The canonicalized owner can sign for its supplier again.
	upperOwnerSupplier, found = k.GetDehydratedSupplier(ctx, upperOwnerOperator)
	require.True(t, found)
	require.Equal(t, upperOwnerOwner, upperOwnerSupplier.OwnerAddress)
	unstakeSupplier(t, k, ctx, upperOwnerOwner, upperOwnerOperator)
}
