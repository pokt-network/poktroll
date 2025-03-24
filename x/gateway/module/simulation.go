package gateway

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/pokt-network/poktroll/testutil/sample"
	gatewaysimulation "github.com/pokt-network/poktroll/x/gateway/simulation"
	"github.com/pokt-network/poktroll/x/gateway/types"
)

// avoid unused import issue
var (
	_ = gatewaysimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgStakeGateway = "op_weight_msg_stake_gateway"
	// TODO_TECHDEBT: Determine the simulation weight value
	defaultWeightMsgStakeGateway int = 100

	opWeightMsgUnstakeGateway = "op_weight_msg_unstake_gateway"
	// TODO_TECHDEBT: Determine the simulation weight value
	defaultWeightMsgUnstakeGateway int = 100

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
	gatewayGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&gatewayGenesis)
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

	var weightMsgStakeGateway int
	simState.AppParams.GetOrGenerate(opWeightMsgStakeGateway, &weightMsgStakeGateway, nil,
		func(_ *rand.Rand) {
			weightMsgStakeGateway = defaultWeightMsgStakeGateway
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeGateway,
		gatewaysimulation.SimulateMsgStakeGateway(am.accountKeeper, am.bankKeeper, am.gatewayKeeper),
	))

	var weightMsgUnstakeGateway int
	simState.AppParams.GetOrGenerate(opWeightMsgUnstakeGateway, &weightMsgUnstakeGateway, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeGateway = defaultWeightMsgUnstakeGateway
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeGateway,
		gatewaysimulation.SimulateMsgUnstakeGateway(am.accountKeeper, am.bankKeeper, am.gatewayKeeper),
	))

	var weightMsgUpdateParam int
	simState.AppParams.GetOrGenerate(opWeightMsgUpdateParam, &weightMsgUpdateParam, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateParam = defaultWeightMsgUpdateParam
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateParam,
		gatewaysimulation.SimulateMsgUpdateParam(am.accountKeeper, am.bankKeeper, am.gatewayKeeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgStakeGateway,
			defaultWeightMsgStakeGateway,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				gatewaysimulation.SimulateMsgStakeGateway(am.accountKeeper, am.bankKeeper, am.gatewayKeeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUnstakeGateway,
			defaultWeightMsgUnstakeGateway,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				gatewaysimulation.SimulateMsgUnstakeGateway(am.accountKeeper, am.bankKeeper, am.gatewayKeeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUpdateParam,
			defaultWeightMsgUpdateParam,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				gatewaysimulation.SimulateMsgUpdateParam(am.accountKeeper, am.bankKeeper, am.gatewayKeeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
