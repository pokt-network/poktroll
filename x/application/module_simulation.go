package application

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"pocket/testutil/sample"
	applicationsimulation "pocket/x/application/simulation"
	"pocket/x/application/types"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = applicationsimulation.FindAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
	_ = rand.Rand{}
)

const (
	opWeightMsgStakeApplication = "op_weight_msg_stake_application"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStakeApplication int = 100

	opWeightMsgUnstakeApplication = "op_weight_msg_unstake_application"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnstakeApplication int = 100

	opWeightMsgDelegateToGateway = "op_weight_msg_delegate_to_gateway"
	// TODO: Determine the simulation weight value
	defaultWeightMsgDelegateToGateway int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	applicationGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&applicationGenesis)
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

	var weightMsgStakeApplication int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStakeApplication, &weightMsgStakeApplication, nil,
		func(_ *rand.Rand) {
			weightMsgStakeApplication = defaultWeightMsgStakeApplication
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeApplication,
		applicationsimulation.SimulateMsgStakeApplication(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnstakeApplication int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnstakeApplication, &weightMsgUnstakeApplication, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeApplication = defaultWeightMsgUnstakeApplication
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeApplication,
		applicationsimulation.SimulateMsgUnstakeApplication(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgDelegateToGateway int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgDelegateToGateway, &weightMsgDelegateToGateway, nil,
		func(_ *rand.Rand) {
			weightMsgDelegateToGateway = defaultWeightMsgDelegateToGateway
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgDelegateToGateway,
		applicationsimulation.SimulateMsgDelegateToGateway(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgStakeApplication,
			defaultWeightMsgStakeApplication,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				applicationsimulation.SimulateMsgStakeApplication(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUnstakeApplication,
			defaultWeightMsgUnstakeApplication,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				applicationsimulation.SimulateMsgUnstakeApplication(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgDelegateToGateway,
			defaultWeightMsgDelegateToGateway,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				applicationsimulation.SimulateMsgDelegateToGateway(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
