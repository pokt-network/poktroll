package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	capMinStake = int64(1_000_000)
	capPrice    = int64(500_000)
)

// enableServiceCap sets min_stake=1 POKT, included=1, price=0.5 POKT.
func enableServiceCap(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context) {
	t.Helper()
	params := k.Keeper.GetParams(ctx)
	minStake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, capMinStake)
	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, capPrice)
	params.MinStake = &minStake
	params.MinStakeIncludedServices = 1
	params.StakePerAdditionalService = &price
	require.NoError(t, k.SetParams(ctx, params))
}

// multiServiceStakeMsg returns an operator-signed stake msg for the given services,
// registering each service first.
func multiServiceStakeMsg(
	t *testing.T,
	k keepertest.SupplierModuleKeepers,
	ctx cosmostypes.Context,
	ownerAddr, operatorAddr string,
	stake int64,
	serviceIds ...string,
) *suppliertypes.MsgStakeSupplier {
	t.Helper()
	for _, serviceId := range serviceIds {
		k.ServiceKeeper.SetService(ctx, sharedtypes.Service{Id: serviceId, ComputeUnitsPerRelay: 1, OwnerAddress: sample.AccAddressBech32()})
	}
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, stake, serviceIds...)
	return stakeMsg
}

func TestStakeSupplier_ServiceCap_RejectsOneOverAndAcceptsAtCap(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	// 1 POKT covers 1 service; 3 services need 2 POKT.
	overMsg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake, "svc1", "svc2", "svc3")
	_, err := srv.StakeSupplier(ctx, overMsg)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())
	require.ErrorContains(t, err, "2000000upokt")

	atCapMsg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+2*capPrice, "svc1", "svc2", "svc3")
	_, err = srv.StakeSupplier(ctx, atCapMsg)
	require.NoError(t, err)
}

func TestStakeSupplier_ServiceCap_DisabledAllowsAnyCount(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	// Default params: price 0, cap disabled.
	msg := multiServiceStakeMsg(t, k, sdkCtx, sample.AccAddressBech32(), sample.AccAddressBech32(), 1_000_000, "svc1", "svc2", "svc3", "svc4")
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)
}

func TestStakeSupplier_ServiceCap_WritesServicePriorityInMessageOrder(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, 1_000_000, "sol", "eth", "base")
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	// Reloaded from the store, not the in-memory supplier.
	stored, found := k.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	require.Equal(t, []string{"sol", "eth", "base"}, stored.ServicePriority)

	// A stake-only update keeps the priority.
	stakeOnly := &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(2_000_000)},
	}
	_, err = srv.StakeSupplier(ctx, stakeOnly)
	require.NoError(t, err)
	stored, _ = k.GetDehydratedSupplier(ctx, operatorAddr)
	require.Equal(t, []string{"sol", "eth", "base"}, stored.ServicePriority)
}

func TestStakeSupplier_ServiceCap_NonCustodialFlow(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	// 1. Owner stakes at min stake with no services.
	ownerMsg := &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	}
	_, err := srv.StakeSupplier(ctx, ownerMsg)
	require.NoError(t, err)

	// 2. Operator sets 2 services without a stake field: rejected, stake covers 1.
	operatorMsg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, 0, "svc1", "svc2")
	operatorMsg.Stake = nil
	setStakeMsgSigner(operatorMsg, operatorAddr)
	_, err = srv.StakeSupplier(ctx, operatorMsg)
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())

	// 3. Owner tops up; the operator retries and succeeds.
	ownerMsg.Stake = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake + capPrice)}
	_, err = srv.StakeSupplier(ctx, ownerMsg)
	require.NoError(t, err)
	operatorMsg.Stake = nil
	_, err = srv.StakeSupplier(ctx, operatorMsg)
	require.NoError(t, err)
}

func TestStakeSupplier_ServiceCap_StakeOnlyDecreaseBelowServicesRejected(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+capPrice, "svc1", "svc2")
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	stakeOnly := &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	}
	_, err = srv.StakeSupplier(ctx, stakeOnly)
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())
}

// A Morse supplier claim stakes through Keeper.StakeSupplier with the claimant
// as signer, owner and operator; the cap applies on that path too.
func TestStakeSupplier_ServiceCap_KeeperLevelMorseStylePath(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	enableServiceCap(t, k, sdkCtx)
	addr := sample.AccAddressBech32()

	msg := multiServiceStakeMsg(t, k, sdkCtx, addr, addr, capMinStake, "svc1", "svc2")
	_, err := k.StakeSupplier(ctx, log.NewNopLogger(), msg)
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())

	msg = multiServiceStakeMsg(t, k, sdkCtx, addr, addr, capMinStake+capPrice, "svc1", "svc2")
	_, err = k.StakeSupplier(ctx, log.NewNopLogger(), msg)
	require.NoError(t, err)
}

func TestStakeSupplier_ServiceCap_StakeOnlyIncreaseAcceptedWhileOverCap(t *testing.T) {
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	enableServiceCap(t, k, sdkCtx)
	ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()

	// 2 services at min+1*price: exactly at the cap.
	msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake+capPrice, "svc1", "svc2")
	_, err := srv.StakeSupplier(ctx, msg)
	require.NoError(t, err)

	// Governance raises the price to 2 POKT: the stake now covers 1 service.
	params := k.Keeper.GetParams(sdkCtx)
	price := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 2_000_000)
	params.StakePerAdditionalService = &price
	require.NoError(t, k.SetParams(sdkCtx, params))

	stakeOnly := func(amount int64) *suppliertypes.MsgStakeSupplier {
		return &suppliertypes.MsgStakeSupplier{
			Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: operatorAddr,
			Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(amount)},
		}
	}

	// A decrease is rejected.
	_, err = srv.StakeSupplier(ctx, stakeOnly(capMinStake))
	require.ErrorContains(t, err, suppliertypes.ErrSupplierTooManyServices.Error())

	// An increase that still covers only 1 service is accepted.
	_, err = srv.StakeSupplier(ctx, stakeOnly(capMinStake+capPrice+1))
	require.NoError(t, err)
}
