package service_test

import (
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	"github.com/pokt-network/poktroll/testutil/sample"
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

// The supplier allowlist and its pending sweeps MUST round-trip. Dropping the
// allowlist silently reopens every permissioned service; dropping a sweep silently
// leaves unlisted suppliers on the service.
func TestGenesis_SupplierAllowlistRoundTrip(t *testing.T) {
	ownerAddr := sample.AccAddressBech32()
	listedAddr1 := sample.AccAddressBech32()
	listedAddr2 := sample.AccAddressBech32()
	services := []sharedtypes.Service{
		{Id: "svc1", Name: "service one", OwnerAddress: ownerAddr, ComputeUnitsPerRelay: 1},
		{Id: "svc2", Name: "service two", OwnerAddress: ownerAddr, ComputeUnitsPerRelay: 1},
		{Id: "svc3", Name: "service three", OwnerAddress: ownerAddr, ComputeUnitsPerRelay: 1},
	}

	k, ctx := keepertest.ServiceKeeper(t)
	for _, svc := range services {
		k.SetService(ctx, svc)
	}
	k.SetSupplierAllowlistEntry(ctx, "svc1", listedAddr1)
	k.SetSupplierAllowlistEntry(ctx, "svc1", listedAddr2)
	k.SetSupplierAllowlistEntry(ctx, "svc2", listedAddr1)
	// A running sweep with a rerun requested, a deferred initial sweep, and a
	// sweep of a service that was reopened since (no entries left).
	k.SetAllowlistSweep(ctx, types.AllowlistSweep{ServiceId: "svc1", Cursor: []byte("supplier-config-key"), Rerun: true})
	k.SetAllowlistSweep(ctx, types.AllowlistSweep{ServiceId: "svc2", DeferredFromHeight: 42})
	k.SetAllowlistSweep(ctx, types.AllowlistSweep{ServiceId: "svc3"})

	exported := service.ExportGenesis(ctx, k)
	require.NotNil(t, exported)
	require.NoError(t, exported.Validate(), "an exported genesis must be importable")
	require.Len(t, exported.SupplierAllowlist, 3)
	require.Len(t, exported.SupplierAllowlistSweeps, 3)

	freshKeeper, freshCtx := keepertest.ServiceKeeper(t)
	service.InitGenesis(freshCtx, freshKeeper, *exported)
	reexported := service.ExportGenesis(freshCtx, freshKeeper)

	require.Equal(t, exported.SupplierAllowlist, reexported.SupplierAllowlist)
	require.Equal(t, exported.SupplierAllowlistSweeps, reexported.SupplierAllowlistSweeps)
	require.Equal(t, k.GetPendingAllowlistSweeps(ctx), freshKeeper.GetPendingAllowlistSweeps(freshCtx))

	require.True(t, freshKeeper.IsServicePermissioned(freshCtx, "svc1"))
	require.True(t, freshKeeper.HasSupplierAllowlistEntry(freshCtx, "svc1", listedAddr1))
	require.True(t, freshKeeper.HasSupplierAllowlistEntry(freshCtx, "svc1", listedAddr2))
	require.True(t, freshKeeper.HasSupplierAllowlistEntry(freshCtx, "svc2", listedAddr1))
	require.False(t, freshKeeper.HasSupplierAllowlistEntry(freshCtx, "svc2", listedAddr2))
	require.False(t, freshKeeper.IsServicePermissioned(freshCtx, "svc3"))
}

// A genesis written before the supplier allowlist existed (no supplier_allowlist /
// supplier_allowlist_sweeps keys) must still validate and import: LocalNet and
// exported networks start from such files.
func TestGenesis_PreAllowlistJSON(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := types.DefaultGenesis()
	genesis.ServiceList = []sharedtypes.Service{
		{Id: "svc1", Name: "service one", OwnerAddress: sample.AccAddressBech32(), ComputeUnitsPerRelay: 1},
	}

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(cdc.MustMarshalJSON(genesis), &fields))
	for _, key := range []string{"supplier_allowlist", "supplier_allowlist_sweeps"} {
		require.Contains(t, fields, key)
		delete(fields, key)
	}
	legacyJSON, err := json.Marshal(fields)
	require.NoError(t, err)

	require.NoError(t, service.AppModuleBasic{}.ValidateGenesis(cdc, nil, legacyJSON))

	var legacyGenesis types.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(legacyJSON, &legacyGenesis))
	k, ctx := keepertest.ServiceKeeper(t)
	service.InitGenesis(ctx, k, legacyGenesis)

	exported := service.ExportGenesis(ctx, k)
	require.Len(t, exported.ServiceList, 1)
	require.Empty(t, exported.SupplierAllowlist)
	require.Empty(t, exported.SupplierAllowlistSweeps)
	require.False(t, k.IsServicePermissioned(ctx, "svc1"))
}
