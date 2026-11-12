package keeper_test

import (
	"fmt"
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

	"github.com/pokt-network/poktroll/testutil/gateway/mocks"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/gateway/keeper"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// A failed stake refund must not fail the gateway unbonding EndBlocker: the
// gateway is removed and its stake stays in the module pool.
func TestEndBlockerUnbondGateways_SendFailureDoesNotHalt(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(gatewaytypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	ctrl := gomock.NewController(t)
	mockBankKeeper := mocks.NewMockBankKeeper(ctrl)
	mockBankKeeper.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), gatewaytypes.ModuleName, gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("simulated send failure")).
		AnyTimes()

	sharedParams := sharedtypes.DefaultParams()
	mockSharedKeeper := mocks.NewMockSharedKeeper(ctrl)
	mockSharedKeeper.EXPECT().GetParams(gomock.Any()).Return(sharedParams).AnyTimes()
	mockSharedKeeper.EXPECT().GetParamsAtHeight(gomock.Any(), gomock.Any()).Return(sharedParams).AnyTimes()

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		mockBankKeeper,
		mockSharedKeeper,
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.SetParams(ctx, gatewaytypes.DefaultParams()))

	// A gateway that began unbonding at session end height 10. With default
	// shared params (NumBlocksPerSession=10, GatewayUnbondingPeriodSessions=1),
	// its unbonding end height is 10 + 10 = 20.
	stake := sdk.NewInt64Coin("upokt", 1000000)
	gw := gatewaytypes.Gateway{
		Address:                 sample.AccAddressBech32(),
		Stake:                   &stake,
		UnstakeSessionEndHeight: 10,
	}
	k.SetGateway(ctx, gw)

	// Run the unbonding EndBlocker at a session-end height past the unbonding end.
	// 20 is a session end height (20 % 10 == 0) and >= unbondingEndHeight (20).
	ctx = ctx.WithBlockHeight(20)
	_, err := k.EndBlockerUnbondGateways(ctx)

	require.NoError(t, err)
	_, found := k.GetGateway(ctx, gw.Address)
	require.False(t, found, "gateway removed even though its refund failed")
}
