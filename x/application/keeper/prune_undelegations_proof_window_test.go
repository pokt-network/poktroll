package keeper_test

import (
	"context"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/testutil/application/mocks"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// A gateway undelegated during a session stays in that session's ring until the
// session's proof window closes, also when the proof window closes later than
// the grace period plus the retention sessions.
func TestPruneUndelegations_KeptUntilProofWindowCloses(t *testing.T) {
	params := sharedtypes.DefaultParams()
	params.NumBlocksPerSession = 10
	params.GracePeriodEndOffsetBlocks = 9
	params.ClaimWindowOpenOffsetBlocks = 11
	params.ClaimWindowCloseOffsetBlocks = 10
	params.ProofWindowOpenOffsetBlocks = 1
	params.ProofWindowCloseOffsetBlocks = 10
	params.ApplicationUnbondingPeriodSessions = 4
	params.SupplierUnbondingPeriodSessions = 4
	params.GatewayUnbondingPeriodSessions = 4
	require.NoError(t, params.ValidateBasic())

	storeKey := storetypes.NewKVStoreKey(apptypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	ctrl := gomock.NewController(t)
	sharedKeeper := mocks.NewMockSharedKeeper(ctrl)
	sharedKeeper.EXPECT().GetParams(gomock.Any()).
		DoAndReturn(func(context.Context) sharedtypes.Params { return params }).AnyTimes()

	k := keeper.NewKeeper(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		sample.AccAddressBech32(),
		mocks.NewMockBankKeeper(ctrl),
		mocks.NewMockAccountKeeper(ctrl),
		mocks.NewMockGatewayKeeper(ctrl),
		sharedKeeper,
	)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// The application undelegated the gateway during the session ending at 100.
	sessionEndHeight := int64(100)
	require.Equal(t, sessionEndHeight, sharedtypes.GetSessionEndHeight(&params, sessionEndHeight))
	gateway := sample.AccAddressBech32()
	app := apptypes.Application{
		Address:                   sample.AccAddressBech32(),
		Stake:                     &apptypes.DefaultMinStake,
		ServiceConfigs:            []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}},
		DelegateeGatewayAddresses: []string{},
		PendingUndelegations: map[uint64]apptypes.UndelegatingGatewayList{
			uint64(sessionEndHeight): {GatewayAddresses: []string{gateway}},
		},
	}
	k.SetApplication(ctx, app)

	proofWindowCloseHeight := sharedtypes.GetProofWindowCloseHeight(&params, sessionEndHeight)
	require.Equal(t, int64(133), proofWindowCloseHeight)

	// Prune at every height up to the proof window close: the gateway stays in the ring.
	for height := sessionEndHeight + 1; height <= proofWindowCloseHeight; height++ {
		require.NoError(t, k.EndBlockerPruneAppToGatewayPendingUndelegation(ctx.WithBlockHeight(height)))
	}
	app, found := k.GetApplication(ctx, app.Address)
	require.True(t, found)
	require.Contains(t, rings.GetRingAddressesAtSessionEndHeight(&app, uint64(sessionEndHeight)), gateway)
}
