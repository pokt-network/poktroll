package supplier

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/pokt-network/poktroll/testutil/sample"
	suppliersimulation "github.com/pokt-network/poktroll/x/supplier/simulation"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

// avoid unused import issue
var (
	_ = suppliersimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgStakeSupplier = "op_weight_msg_stake_supplier"
	// TODO_TECHDEBT: Determine the simulation weight value
	defaultWeightMsgStakeSupplier int = 100

	opWeightMsgUnstakeSupplier = "op_weight_msg_unstake_supplier"
	// TODO_TECHDEBT: Determine the simulation weight value
	defaultWeightMsgUnstakeSupplier int = 100

	opWeightMsgUpdateParam = "op_weight_msg_update_param"
	// TODO_TECHDEBT: Determine the simulation weight value
	defaultWeightMsgUpdateParam int = 100

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
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgStakeSupplier int
	simState.AppParams.GetOrGenerate(opWeightMsgStakeSupplier, &weightMsgStakeSupplier, nil,
		func(_ *rand.Rand) {
			weightMsgStakeSupplier = defaultWeightMsgStakeSupplier
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeSupplier,
		suppliersimulation.SimulateMsgStakeSupplier(am.accountKeeper, am.bankKeeper, am.supplierKeeper),
	))

	var weightMsgUnstakeSupplier int
	simState.AppParams.GetOrGenerate(opWeightMsgUnstakeSupplier, &weightMsgUnstakeSupplier, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeSupplier = defaultWeightMsgUnstakeSupplier
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeSupplier,
		suppliersimulation.SimulateMsgUnstakeSupplier(am.accountKeeper, am.bankKeeper, am.supplierKeeper),
	))

	var weightMsgUpdateParam int
	simState.AppParams.GetOrGenerate(opWeightMsgUpdateParam, &weightMsgUpdateParam, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateParam = defaultWeightMsgUpdateParam
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateParam,
		suppliersimulation.SimulateMsgUpdateParam(am.accountKeeper, am.bankKeeper, am.supplierKeeper),
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
				suppliersimulation.SimulateMsgStakeSupplier(am.accountKeeper, am.bankKeeper, am.supplierKeeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUnstakeSupplier,
			defaultWeightMsgUnstakeSupplier,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				suppliersimulation.SimulateMsgUnstakeSupplier(am.accountKeeper, am.bankKeeper, am.supplierKeeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUpdateParam,
			defaultWeightMsgUpdateParam,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				suppliersimulation.SimulateMsgUpdateParam(am.accountKeeper, am.bankKeeper, am.supplierKeeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
