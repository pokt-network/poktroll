package keeper_test

import (
	"strings"
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TestMigrateNonCanonicalApplications runs the migration at height 25 (session end 30)
// with default shared params, then the transfer and unbonding EndBlockers at every
// session end until each transfer (40) and unbonding (40 or 50) has completed. It uses
// real bank and shared keepers so refunds land on the decoded account's balance.
func TestMigrateNonCanonicalApplications(t *testing.T) {
	const (
		migrationHeight         = 25
		migrationSessionEnd     = 30
		presetUnstakeSessionEnd = 20
	)
	stake := sdk.NewInt64Coin(pocket.DenomuPOKT, apptypes.DefaultMinStake.Amount.Int64())
	services := []*sharedtypes.ApplicationServiceConfig{{ServiceId: "svc1"}}
	newApp := func(address string, unstakeSessionEndHeight uint64, transferDst string) apptypes.Application {
		app := apptypes.Application{Address: address, Stake: &stake, ServiceConfigs: services, UnstakeSessionEndHeight: unstakeSessionEndHeight}
		if transferDst != "" {
			app.PendingTransfer = &apptypes.PendingApplicationTransfer{DestinationAddress: transferDst, SessionEndHeight: migrationSessionEnd}
		}
		return app
	}

	address, dstAddress := sample.AccAddressBech32(), sample.AccAddressBech32()
	upper, upperDst := strings.ToUpper(address), strings.ToUpper(dstAddress)

	dstApp := newApp(dstAddress, apptypes.ApplicationNotUnstaking, "")

	tests := []struct {
		desc        string
		app         apptypes.Application
		existingApp *apptypes.Application

		expectedNumUnbondingBegun                    int
		expectedNumTransferDestinationsCanonicalized int
		expectedNumTransfersDropped                  int
		// State right after the migration.
		expectedUnstakeSessionEndHeight uint64
		expectedTransferDst             string
		// State after the EndBlockers: remaining application stakes and refunded balances.
		expectedFinalStakes map[string]int64
		expectedRefund      int64
	}{
		{
			desc:                            "non-canonical application begins unbonding and is refunded",
			app:                             newApp(upper, apptypes.ApplicationNotUnstaking, ""),
			expectedNumUnbondingBegun:       1,
			expectedUnstakeSessionEndHeight: migrationSessionEnd,
			expectedFinalStakes:             map[string]int64{},
			expectedRefund:                  stake.Amount.Int64(),
		},
		{
			desc:                            "non-canonical application with a pending transfer: transfer dropped, unbonding begun",
			app:                             newApp(upper, apptypes.ApplicationNotUnstaking, dstAddress),
			expectedNumUnbondingBegun:       1,
			expectedNumTransfersDropped:     1,
			expectedUnstakeSessionEndHeight: migrationSessionEnd,
			expectedFinalStakes:             map[string]int64{},
			expectedRefund:                  stake.Amount.Int64(),
		},
		{
			desc:                            "already unbonding non-canonical application keeps its unbonding",
			app:                             newApp(upper, presetUnstakeSessionEnd, ""),
			expectedUnstakeSessionEndHeight: presetUnstakeSessionEnd,
			expectedFinalStakes:             map[string]int64{},
			expectedRefund:                  stake.Amount.Int64(),
		},
		{
			desc:                "canonical application is untouched",
			app:                 newApp(address, apptypes.ApplicationNotUnstaking, ""),
			expectedFinalStakes: map[string]int64{address: stake.Amount.Int64()},
		},
		{
			desc: "non-canonical transfer destination is canonicalized and the transfer completes to it",
			app:  newApp(address, apptypes.ApplicationNotUnstaking, upperDst),
			expectedNumTransferDestinationsCanonicalized: 1,
			expectedTransferDst:                          dstAddress,
			expectedFinalStakes:                          map[string]int64{dstAddress: stake.Amount.Int64()},
		},
		{
			desc:                        "transfer to the source's own non-canonical address is dropped",
			app:                         newApp(address, apptypes.ApplicationNotUnstaking, upper),
			expectedNumTransfersDropped: 1,
			expectedFinalStakes:         map[string]int64{address: stake.Amount.Int64()},
		},
		{
			desc:                        "transfer to the non-canonical address of an existing application is dropped",
			app:                         newApp(address, apptypes.ApplicationNotUnstaking, upperDst),
			existingApp:                 &dstApp,
			expectedNumTransfersDropped: 1,
			expectedFinalStakes:         map[string]int64{address: stake.Amount.Int64(), dstAddress: stake.Amount.Int64()},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			keepers, ctx := keepertest.NewTokenomicsModuleKeepers(t, log.NewNopLogger())
			k := *keepers.ApplicationKeeper.(*appkeeper.Keeper)
			sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockHeight(migrationHeight)
			require.NoError(t, keepers.MintCoins(sdkCtx, apptypes.ModuleName, sdk.NewCoins(stake)))
			k.SetApplication(sdkCtx, test.app)
			if test.existingApp != nil {
				require.NoError(t, keepers.MintCoins(sdkCtx, apptypes.ModuleName, sdk.NewCoins(stake)))
				k.SetApplication(sdkCtx, *test.existingApp)
			}

			ctx, sdkCtx = testevents.ResetEventManager(sdkCtx)
			numUnbondingBegun, numTransferDestinationsCanonicalized, numTransfersDropped := k.MigrateNonCanonicalApplications(ctx)
			require.Equal(t, test.expectedNumUnbondingBegun, numUnbondingBegun)
			require.Equal(t, test.expectedNumTransferDestinationsCanonicalized, numTransferDestinationsCanonicalized)
			require.Equal(t, test.expectedNumTransfersDropped, numTransfersDropped)
			unbondingBeginEvents := testevents.FilterEvents[*apptypes.EventApplicationUnbondingBegin](t, sdkCtx.EventManager().Events())
			require.Len(t, unbondingBeginEvents, test.expectedNumUnbondingBegun)
			transferErrorEvents := testevents.FilterEvents[*apptypes.EventTransferError](t, sdkCtx.EventManager().Events())
			require.Len(t, transferErrorEvents, test.expectedNumTransfersDropped)
			for _, transferErrorEvent := range transferErrorEvents {
				require.Equal(t, test.app.Address, transferErrorEvent.GetSourceAddress())
				require.Equal(t, test.app.GetPendingTransfer().GetDestinationAddress(), transferErrorEvent.GetDestinationAddress())
			}

			app, found := k.GetApplication(sdkCtx, test.app.Address)
			require.True(t, found)
			require.Equal(t, test.expectedUnstakeSessionEndHeight, app.UnstakeSessionEndHeight)
			require.Equal(t, test.expectedTransferDst, app.GetPendingTransfer().GetDestinationAddress())

			for height := int64(migrationSessionEnd); height <= 50; height += int64(sharedtypes.DefaultNumBlocksPerSession) {
				endBlockCtx := sdkCtx.WithBlockHeight(height)
				require.NoError(t, k.EndBlockerTransferApplication(endBlockCtx))
				require.NoError(t, k.EndBlockerUnbondApplications(endBlockCtx))
			}

			finalStakes := make(map[string]int64)
			for _, finalApp := range k.GetAllApplications(sdkCtx) {
				finalStakes[finalApp.Address] = finalApp.GetStake().Amount.Int64()
			}
			require.Equal(t, test.expectedFinalStakes, finalStakes)

			balance, err := keepers.Balance(sdkCtx, &banktypes.QueryBalanceRequest{Address: address, Denom: pocket.DenomuPOKT})
			require.NoError(t, err)
			require.Equal(t, test.expectedRefund, balance.GetBalance().Amount.Int64(), "refund goes to the canonical account")
		})
	}
}
