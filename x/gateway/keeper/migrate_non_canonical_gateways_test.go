package keeper_test

import (
	"context"
	"strings"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	testevents "github.com/pokt-network/poktroll/testutil/events"
	"github.com/pokt-network/poktroll/testutil/gateway/mocks"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestMigrateNonCanonicalGateways runs the migration at height 25 (session end 30)
// with default shared params, then the unbonding EndBlocker at every session end
// until each unbonding (30 or 40) has completed. Refunds are recorded by the
// decoded recipient address.
func TestMigrateNonCanonicalGateways(t *testing.T) {
	const (
		migrationHeight         = 25
		migrationSessionEnd     = 30
		presetUnstakeSessionEnd = 20
	)
	stake := sdk.NewInt64Coin("upokt", 1_000_000)
	metadata := &sharedtypes.Metadata{Card: []byte(`{"name":"gw"}`)}
	address := sample.AccAddressBech32()
	upper := strings.ToUpper(address)

	tests := []struct {
		desc    string
		gateway gatewaytypes.Gateway

		expectedNumUnbondingBegun       int
		expectedUnstakeSessionEndHeight uint64
		// State after the EndBlockers.
		expectedRemoved bool
		expectedRefunds map[string]int64
	}{
		{
			desc:                            "non-canonical gateway begins unbonding and is refunded",
			gateway:                         gatewaytypes.Gateway{Address: upper, Stake: &stake, Metadata: metadata},
			expectedNumUnbondingBegun:       1,
			expectedUnstakeSessionEndHeight: migrationSessionEnd,
			expectedRemoved:                 true,
			expectedRefunds:                 map[string]int64{address: stake.Amount.Int64()},
		},
		{
			desc:                            "already unbonding non-canonical gateway keeps its unbonding",
			gateway:                         gatewaytypes.Gateway{Address: upper, Stake: &stake, UnstakeSessionEndHeight: presetUnstakeSessionEnd},
			expectedUnstakeSessionEndHeight: presetUnstakeSessionEnd,
			expectedRemoved:                 true,
			expectedRefunds:                 map[string]int64{address: stake.Amount.Int64()},
		},
		{
			desc:            "canonical gateway is untouched",
			gateway:         gatewaytypes.Gateway{Address: address, Stake: &stake, Metadata: metadata},
			expectedRefunds: map[string]int64{},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			k, ctx, refunds := newGatewayKeeperRecordingRefunds(t)
			sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(migrationHeight)
			k.SetGateway(sdkCtx, test.gateway)

			ctx, sdkCtx = testevents.ResetEventManager(sdkCtx)
			require.Equal(t, test.expectedNumUnbondingBegun, k.MigrateNonCanonicalGateways(ctx))
			unbondingBeginEvents := testevents.FilterEvents[*gatewaytypes.EventGatewayUnbondingBegin](t, sdkCtx.EventManager().Events())
			require.Len(t, unbondingBeginEvents, test.expectedNumUnbondingBegun)

			gateway, found := k.GetGateway(sdkCtx, test.gateway.Address)
			require.True(t, found)
			require.Equal(t, test.expectedUnstakeSessionEndHeight, gateway.UnstakeSessionEndHeight)
			require.Equal(t, test.gateway.Metadata, gateway.Metadata, "card preserved")

			for height := int64(migrationSessionEnd); height <= 40; height += int64(sharedtypes.DefaultNumBlocksPerSession) {
				_, err := k.EndBlockerUnbondGateways(sdkCtx.WithBlockHeight(height))
				require.NoError(t, err)
			}

			_, found = k.GetGateway(sdkCtx, test.gateway.Address)
			require.Equal(t, test.expectedRemoved, !found)
			require.Equal(t, test.expectedRefunds, refunds)
		})
	}
}

// TestMigrateNonCanonicalGateways_AutoUndelegates asserts that an application
// delegated to a non-canonical gateway is undelegated from it once the migrated
// gateway is no longer active.
func TestMigrateNonCanonicalGateways_AutoUndelegates(t *testing.T) {
	keepers, ctx := keepertest.NewTokenomicsModuleKeepers(t, log.NewNopLogger())
	appKeeper := *keepers.ApplicationKeeper.(*appkeeper.Keeper)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(25)

	stake := sdk.NewInt64Coin("upokt", 1_000_000)
	upperGateway := strings.ToUpper(sample.AccAddressBech32())
	keepers.GatewayKeeper.SetGateway(sdkCtx, gatewaytypes.Gateway{Address: upperGateway, Stake: &stake})
	appAddress := sample.AccAddressBech32()
	appKeeper.SetApplication(sdkCtx, apptypes.Application{
		Address:                   appAddress,
		Stake:                     &stake,
		ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
		DelegateeGatewayAddresses: []string{upperGateway},
	})

	require.Equal(t, 1, keepers.GatewayKeeper.MigrateNonCanonicalGateways(sdkCtx))

	// The gateway stays active until the end of the session (30).
	require.NoError(t, appKeeper.EndBlockerAutoUndelegateFromUnbondingGateways(sdkCtx.WithBlockHeight(31)))
	app, found := appKeeper.GetApplication(sdkCtx, appAddress)
	require.True(t, found)
	require.Empty(t, app.DelegateeGatewayAddresses)
	delegationIterator := appKeeper.GetDelegationsIterator(sdkCtx, upperGateway)
	defer delegationIterator.Close()
	require.False(t, delegationIterator.Valid(), "delegation index entry removed")
}

// newGatewayKeeperRecordingRefunds returns a gateway keeper with a real shared
// keeper on the same store and a bank keeper that records each refund by the
// recipient's address string.
func newGatewayKeeperRecordingRefunds(t *testing.T) (keeper.Keeper, context.Context, map[string]int64) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(gatewaytypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	sharedKeeper := sharedkeeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), log.NewNopLogger(), authority)
	require.NoError(t, sharedKeeper.SetParams(ctx, sharedtypes.DefaultParams()))

	refunds := make(map[string]int64)
	mockBankKeeper := mocks.NewMockBankKeeper(gomock.NewController(t))
	mockBankKeeper.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), gatewaytypes.ModuleName, gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, _ string, addr sdk.AccAddress, coins sdk.Coins) {
			refunds[addr.String()] += coins[0].Amount.Int64()
		}).
		AnyTimes()

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), log.NewNopLogger(), authority, mockBankKeeper, sharedKeeper)
	require.NoError(t, k.SetParams(ctx, gatewaytypes.DefaultParams()))
	return k, ctx, refunds
}
