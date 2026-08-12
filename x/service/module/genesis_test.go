package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	service "github.com/pokt-network/poktroll/x/service/module"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		ServiceList: []sharedtypes.Service{
			{
				Id:   "svc1",
				Name: "service one",
			},
			{
				Id:   "svc2",
				Name: "service two",
			},
		},

		RelayMiningDifficultyList: []types.RelayMiningDifficulty{
			{
				ServiceId: "0",
			},
			{
				ServiceId: "1",
			},
		},
		// The cupr history MUST round-trip. If it is dropped on export/import, every past
		// session silently resolves to the LIVE cupr instead of the value it was mined
		// under -- the exact regression the session-start pin exists to prevent, with no
		// error to notice.
		ComputeUnitsPerRelayHistory: []types.ServiceComputeUnitsPerRelayUpdate{
			{
				ServiceId:            "svc1",
				EffectiveHeight:      1,
				ComputeUnitsPerRelay: 7,
			},
			{
				ServiceId:            "svc1",
				EffectiveHeight:      100,
				ComputeUnitsPerRelay: 42,
			},
			{
				ServiceId:            "svc2",
				EffectiveHeight:      50,
				ComputeUnitsPerRelay: 13,
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ServiceKeeper(t)
	service.InitGenesis(ctx, k, genesisState)
	got := service.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.ServiceList, got.ServiceList)
	require.ElementsMatch(t, genesisState.RelayMiningDifficultyList, got.RelayMiningDifficultyList)
	require.ElementsMatch(t, genesisState.ComputeUnitsPerRelayHistory, got.ComputeUnitsPerRelayHistory,
		"cupr history must survive an export/import round trip; dropping it silently reverts "+
			"every past session to the live cupr")

	// The restored entries must be resolvable at-height, not merely present in the export:
	// this is what settlement and x/proof actually call.
	cupr, found := k.GetServiceComputeUnitsPerRelayAtHeight(ctx, "svc1", 99)
	require.True(t, found)
	require.EqualValues(t, 7, cupr, "height 99 falls under the entry effective at height 1")

	cupr, found = k.GetServiceComputeUnitsPerRelayAtHeight(ctx, "svc1", 100)
	require.True(t, found)
	require.EqualValues(t, 42, cupr, "height 100 is covered by the entry effective at height 100")
	// this line is used by starport scaffolding # genesis/test/assert
}
