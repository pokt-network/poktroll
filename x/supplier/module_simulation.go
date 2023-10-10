package supplier

import (
	"math/rand"

	"pocket/testutil/sample"
	suppliersimulation "pocket/x/supplier/simulation"
	"pocket/x/supplier/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = suppliersimulation.FindAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
	_ = rand.Rand{}
)

const (
	opWeightMsgStakeSupplier = "op_weight_msg_stake_supplier"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStakeSupplier int = 100

	opWeightMsgUnstakeSupplier = "op_weight_msg_unstake_supplier"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnstakeSupplier int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	supplierGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&supplierGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgStakeSupplier int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStakeSupplier, &weightMsgStakeSupplier, nil,
		func(_ *rand.Rand) {
			weightMsgStakeSupplier = defaultWeightMsgStakeSupplier
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeSupplier,
		suppliersimulation.SimulateMsgStakeSupplier(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnstakeSupplier int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnstakeSupplier, &weightMsgUnstakeSupplier, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeSupplier = defaultWeightMsgUnstakeSupplier
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeSupplier,
		suppliersimulation.SimulateMsgUnstakeSupplier(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgStakeSupplier,
			defaultWeightMsgStakeSupplier,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				suppliersimulation.SimulateMsgStakeSupplier(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUnstakeSupplier,
			defaultWeightMsgUnstakeSupplier,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				suppliersimulation.SimulateMsgUnstakeSupplier(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
