//go:build test

package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func TestRecoverSupplierOwners(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "refund_to_new_owner", true: "cancel_unbonding"}[cancel], func(t *testing.T) {
			k, ctx := keepertest.SupplierKeeper(t)
			srv := keeper.NewMsgServerImpl(*k.Keeper)
			sharedParams := k.SharedKeeper.GetParams(ctx)
			stake := suppliertypes.DefaultMinStake.Amount.Int64()

			owner, operator, otherOwner, newOwner := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

			// Stake, then an owner-signed stake-only update moving the owner to otherOwner, then unstake by otherOwner.
			stakeMsg, _ := newSupplierStakeMsg(owner, operator, stake, serviceID)
			_, err := srv.StakeSupplier(ctx, stakeMsg)
			require.NoError(t, err)
			// Stake-only (no services): an owner-signed message may not touch service configs.
			ownerChangeMsg, _ := newSupplierStakeMsg(otherOwner, operator, stake, serviceID)
			ownerChangeMsg.Signer = owner
			ownerChangeMsg.Services = nil
			_, err = srv.StakeSupplier(ctx, ownerChangeMsg)
			require.NoError(t, err)
			_, err = srv.UnstakeSupplier(ctx, &suppliertypes.MsgUnstakeSupplier{Signer: otherOwner, OperatorAddress: operator})
			require.NoError(t, err)
			unbonding, found := k.GetDehydratedSupplier(ctx, operator)
			require.True(t, found)
			require.Equal(t, otherOwner, unbonding.OwnerAddress)
			require.True(t, unbonding.IsUnbonding())
			unbondingEndHeight := k.GetSupplierUnbondingEndHeight(ctx, &unbonding)

			ctx, _ = testutilevents.ResetEventManager(ctx)
			recoveries := []keeper.SupplierOwnerRecovery{
				{OperatorAddress: operator, NewOwnerAddress: newOwner},
				{OperatorAddress: sample.AccAddressBech32(), NewOwnerAddress: newOwner},                           // unknown operator
				{OperatorAddress: operator, NewOwnerAddress: ""},                                                  // placeholder left empty
				{OperatorAddress: operator, NewOwnerAddress: authtypes.NewModuleAddress("distribution").String()}, // bank-blocked recipient
			}
			numRecovered, numSkipped := k.RecoverSupplierOwners(ctx, recoveries, cancel)
			require.Equal(t, 1, numRecovered)
			require.Equal(t, 3, numSkipped)

			recovered, found := k.GetDehydratedSupplier(ctx, operator)
			require.True(t, found)
			require.Equal(t, newOwner, recovered.OwnerAddress)
			require.Equal(t, !cancel, recovered.IsUnbonding())
			events := testutilevents.FilterEvents[*suppliertypes.EventSupplierUnbondingCanceled](t, cosmostypes.UnwrapSDKContext(ctx).EventManager().Events())
			if cancel {
				require.Len(t, events, 1)
				require.Equal(t, newOwner, events[0].Supplier.OwnerAddress)
			} else {
				require.Empty(t, events)
			}

			// Run the unbonding EndBlocker at the original unbonding end height.
			ctx = keepertest.SetBlockHeight(ctx, sharedtypes.GetSessionEndHeight(&sharedParams, unbondingEndHeight))
			numUnbonded, err := k.EndBlockerUnbondSuppliers(ctx)
			require.NoError(t, err)
			if cancel {
				require.Zero(t, numUnbonded)
				_, found = k.GetDehydratedSupplier(ctx, operator)
				require.True(t, found, "cancelled unbonding keeps the supplier staked")
				require.Zero(t, k.SupplierBalanceMap[newOwner])
			} else {
				require.EqualValues(t, 1, numUnbonded)
				_, found = k.GetDehydratedSupplier(ctx, operator)
				require.False(t, found)
				require.Equal(t, stake, k.SupplierBalanceMap[newOwner], "refund lands on the new owner")
			}
			require.Zero(t, k.SupplierBalanceMap[otherOwner], "the previous owner never receives the stake")
		})
	}
}

// TestRecoverSupplierOwners_MultisigOwner uses a 2-of-3 legacy multisig account as the
// recovery destination: the handler accepts it and the unbonding refund lands on it.
func TestRecoverSupplierOwners_MultisigOwner(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	sharedParams := k.SharedKeeper.GetParams(ctx)
	stake := suppliertypes.DefaultMinStake.Amount.Int64()

	multisigPubKey := multisig.NewLegacyAminoPubKey(2, []cryptotypes.PubKey{
		secp256k1.GenPrivKey().PubKey(), secp256k1.GenPrivKey().PubKey(), secp256k1.GenPrivKey().PubKey(),
	})
	multisigOwner := cosmostypes.AccAddress(multisigPubKey.Address()).String()
	require.NoError(t, sharedtypes.ValidateCanonicalAddress(multisigOwner))

	otherOwner, operator := sample.AccAddressBech32(), sample.AccAddressBech32()
	stakeMsg, _ := newSupplierStakeMsg(otherOwner, operator, stake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)
	_, err = srv.UnstakeSupplier(ctx, &suppliertypes.MsgUnstakeSupplier{Signer: otherOwner, OperatorAddress: operator})
	require.NoError(t, err)
	unbonding, _ := k.GetDehydratedSupplier(ctx, operator)
	unbondingEndHeight := k.GetSupplierUnbondingEndHeight(ctx, &unbonding)

	numRecovered, numSkipped := k.RecoverSupplierOwners(ctx, []keeper.SupplierOwnerRecovery{{OperatorAddress: operator, NewOwnerAddress: multisigOwner}}, false)
	require.Equal(t, 1, numRecovered)
	require.Zero(t, numSkipped)

	ctx = keepertest.SetBlockHeight(ctx, sharedtypes.GetSessionEndHeight(&sharedParams, unbondingEndHeight))
	numUnbonded, err := k.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, numUnbonded)
	require.Equal(t, stake, k.SupplierBalanceMap[multisigOwner])
	require.Zero(t, k.SupplierBalanceMap[otherOwner])
}

// TestRecoverSupplierOwners_StakedWithHostileOperator covers a supplier whose unbonding
// was cancelled by the other owner before the upgrade, so it is staked (not unbonding)
// when recovered, and whose operator key stays in the other party's hands afterwards.
// The recovery begins a voluntary unbonding under the new owner; neither the previous
// owner nor the operator can cancel it, and the refund lands on the new owner.
func TestRecoverSupplierOwners_StakedWithHostileOperator(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	sharedParams := k.SharedKeeper.GetParams(ctx)
	stake := suppliertypes.DefaultMinStake.Amount.Int64()
	owner, operator, otherOwner, newOwner := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

	stakeMsg, _ := newSupplierStakeMsg(owner, operator, stake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)
	ownerChangeMsg, _ := newSupplierStakeMsg(otherOwner, operator, stake)
	ownerChangeMsg.Signer = owner
	_, err = srv.StakeSupplier(ctx, ownerChangeMsg)
	require.NoError(t, err)
	_, err = srv.UnstakeSupplier(ctx, &suppliertypes.MsgUnstakeSupplier{Signer: otherOwner, OperatorAddress: operator})
	require.NoError(t, err)
	// The other owner cancels the unbonding, then the operator re-adds a service with
	// a revenue share to the other party: staked again, serving.
	cancelMsg, _ := newSupplierStakeMsg(otherOwner, operator, stake)
	cancelMsg.Signer = otherOwner
	_, err = srv.StakeSupplier(ctx, cancelMsg)
	require.NoError(t, err)
	servicesMsg, _ := newSupplierStakeMsg(otherOwner, operator, stake, serviceID)
	servicesMsg.Signer = operator
	_, err = srv.StakeSupplier(ctx, servicesMsg)
	require.NoError(t, err)
	staked, found := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, found)
	require.False(t, staked.IsUnbonding())

	ctx, _ = testutilevents.ResetEventManager(ctx)
	numRecovered, numSkipped := k.RecoverSupplierOwners(ctx, []keeper.SupplierOwnerRecovery{{OperatorAddress: operator, NewOwnerAddress: newOwner}}, false)
	require.Equal(t, 1, numRecovered)
	require.Zero(t, numSkipped)
	recovered, _ := k.GetDehydratedSupplier(ctx, operator)
	require.Equal(t, newOwner, recovered.OwnerAddress)
	require.True(t, recovered.IsUnbonding(), "a listed supplier found staked begins unbonding")
	require.EqualValues(t, sharedtypes.GetSessionEndHeight(&sharedParams, cosmostypes.UnwrapSDKContext(ctx).BlockHeight()), recovered.UnstakeSessionEndHeight)
	beginEvents := testutilevents.FilterEvents[*suppliertypes.EventSupplierUnbondingBegin](t, cosmostypes.UnwrapSDKContext(ctx).EventManager().Events())
	require.Len(t, beginEvents, 1)
	require.Equal(t, newOwner, beginEvents[0].Supplier.OwnerAddress)
	require.Equal(t, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_VOLUNTARY, beginEvents[0].Reason)

	// Neither the previous owner nor the operator can cancel the unbonding or act as owner.
	lockedOut, _ := newSupplierStakeMsg(otherOwner, operator, stake)
	lockedOut.Signer = otherOwner
	_, err = srv.StakeSupplier(ctx, lockedOut)
	require.ErrorContains(t, err, "unauthorized supplier update")
	operatorCancel, _ := newSupplierStakeMsg(newOwner, operator, stake, serviceID)
	operatorCancel.Signer = operator
	_, err = srv.StakeSupplier(ctx, operatorCancel)
	require.Error(t, err, "operator must not be able to cancel the unbonding")
	_, err = srv.UnstakeSupplier(ctx, &suppliertypes.MsgUnstakeSupplier{Signer: otherOwner, OperatorAddress: operator})
	require.Error(t, err)

	unbonding, _ := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, unbonding.IsUnbonding())
	ctx = keepertest.SetBlockHeight(ctx, sharedtypes.GetSessionEndHeight(&sharedParams, k.GetSupplierUnbondingEndHeight(ctx, &unbonding)))
	numUnbonded, err := k.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, numUnbonded)
	require.Equal(t, stake, k.SupplierBalanceMap[newOwner])
	// The other owner only ever paid the staking fee of its cancel; it never receives funds.
	require.LessOrEqual(t, k.SupplierBalanceMap[otherOwner], int64(0))
}

// TestRecoverSupplierOwners_BelowMinStakeUnbonding covers a supplier whose current
// unbonding was started by settlement because a slash took its stake below min stake.
// That unbonding carries a marker letting the operator cancel it by restoring min stake.
// The recovery clears the marker, so the operator can no longer cancel and the refund
// (at the original end height) lands on the new owner.
func TestRecoverSupplierOwners_BelowMinStakeUnbonding(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	sharedParams := k.SharedKeeper.GetParams(ctx)
	minStake := suppliertypes.DefaultMinStake.Amount.Int64()
	operator, otherOwner, newOwner := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

	stakeMsg, _ := newSupplierStakeMsg(otherOwner, operator, minStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Slash below min stake and start the below-min-stake unbonding exactly as
	// settlement does.
	supplier, found := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, found)
	slashed := suppliertypes.DefaultMinStake.SubAmount(suppliertypes.DefaultMinStake.Amount.QuoRaw(2))
	supplier.Stake = &slashed
	k.SetDehydratedSupplier(ctx, supplier)
	k.BeginSupplierUnbonding(ctx, &supplier, &sharedParams, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE)
	endHeight := k.GetSupplierUnbondingEndHeight(ctx, &supplier)

	numRecovered, numSkipped := k.RecoverSupplierOwners(ctx, []keeper.SupplierOwnerRecovery{{OperatorAddress: operator, NewOwnerAddress: newOwner}}, false)
	require.Equal(t, 1, numRecovered)
	require.Zero(t, numSkipped)
	recovered, _ := k.GetDehydratedSupplier(ctx, operator)
	require.Equal(t, newOwner, recovered.OwnerAddress)
	require.True(t, recovered.IsUnbonding())
	require.EqualValues(t, supplier.UnstakeSessionEndHeight, recovered.UnstakeSessionEndHeight, "the unbonding in progress is kept")

	// The operator restores min stake: without the marker this is no longer a rescue.
	rescueMsg, _ := newSupplierStakeMsg(newOwner, operator, minStake)
	rescueMsg.Signer = operator
	_, err = srv.StakeSupplier(ctx, rescueMsg)
	require.Error(t, err, "operator must not be able to cancel the unbonding of a recovered supplier")
	stillUnbonding, _ := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, stillUnbonding.IsUnbonding())

	ctx = keepertest.SetBlockHeight(ctx, sharedtypes.GetSessionEndHeight(&sharedParams, endHeight))
	numUnbonded, err := k.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, numUnbonded)
	require.Equal(t, slashed.Amount.Int64(), k.SupplierBalanceMap[newOwner], "the slashed stake is refunded to the new owner")
	require.LessOrEqual(t, k.SupplierBalanceMap[otherOwner], int64(0))
}

// TestRecoverSupplierOwners_AtUnbondingEndHeight recovers in the same block in which the
// unbonding ends (upgrade handler in BeginBlock, refund in EndBlock): the refund already
// lands on the new owner.
func TestRecoverSupplierOwners_AtUnbondingEndHeight(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	sharedParams := k.SharedKeeper.GetParams(ctx)
	stake := suppliertypes.DefaultMinStake.Amount.Int64()
	operator, otherOwner, newOwner := sample.AccAddressBech32(), sample.AccAddressBech32(), sample.AccAddressBech32()

	stakeMsg, _ := newSupplierStakeMsg(otherOwner, operator, stake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)
	_, err = srv.UnstakeSupplier(ctx, &suppliertypes.MsgUnstakeSupplier{Signer: otherOwner, OperatorAddress: operator})
	require.NoError(t, err)
	unbonding, _ := k.GetDehydratedSupplier(ctx, operator)
	endHeight := sharedtypes.GetSessionEndHeight(&sharedParams, k.GetSupplierUnbondingEndHeight(ctx, &unbonding))

	ctx = keepertest.SetBlockHeight(ctx, endHeight)
	numRecovered, _ := k.RecoverSupplierOwners(ctx, []keeper.SupplierOwnerRecovery{{OperatorAddress: operator, NewOwnerAddress: newOwner}}, false)
	require.Equal(t, 1, numRecovered)
	numUnbonded, err := k.EndBlockerUnbondSuppliers(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, numUnbonded)
	require.Equal(t, stake, k.SupplierBalanceMap[newOwner])
	require.Zero(t, k.SupplierBalanceMap[otherOwner])
}
