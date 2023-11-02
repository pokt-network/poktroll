package gateway

import (
	"math/rand"

	"github.com/pokt-network/poktroll/testutil/sample"
	gatewaysimulation "github.com/pokt-network/poktroll/x/gateway/simulation"
	"github.com/pokt-network/poktroll/x/gateway/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// avoid unused import issue
var (
	_ = sample.AccAddress
	_ = gatewaysimulation.FindAccount
	_ = simulation.MsgEntryKind
	_ = baseapp.Paramspace
	_ = rand.Rand{}
)

const (
	opWeightMsgStakeGateway = "op_weight_msg_stake_gateway"
	// TODO: Determine the simulation weight value
	defaultWeightMsgStakeGateway int = 100

	opWeightMsgUnstakeGateway = "op_weight_msg_unstake_gateway"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUnstakeGateway int = 100

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
func (am AppModule) RegisterStoreDecoder(_ sdk.StoreDecoderRegistry) {}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(_ module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgStakeGateway int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgStakeGateway, &weightMsgStakeGateway, nil,
		func(_ *rand.Rand) {
			weightMsgStakeGateway = defaultWeightMsgStakeGateway
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgStakeGateway,
		gatewaysimulation.SimulateMsgStakeGateway(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgUnstakeGateway int
	simState.AppParams.GetOrGenerate(simState.Cdc, opWeightMsgUnstakeGateway, &weightMsgUnstakeGateway, nil,
		func(_ *rand.Rand) {
			weightMsgUnstakeGateway = defaultWeightMsgUnstakeGateway
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUnstakeGateway,
		gatewaysimulation.SimulateMsgUnstakeGateway(am.accountKeeper, am.bankKeeper, am.keeper),
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
				gatewaysimulation.SimulateMsgStakeGateway(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUnstakeGateway,
			defaultWeightMsgUnstakeGateway,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				gatewaysimulation.SimulateMsgUnstakeGateway(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
