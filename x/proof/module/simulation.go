package proof

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/pokt-network/pocket/testutil/sample"
	proofsimulation "github.com/pokt-network/pocket/x/proof/simulation"
	"github.com/pokt-network/pocket/x/proof/types"
)

// avoid unused import issue
var (
	_ = proofsimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgCreateClaim = "op_weight_msg_create_claim"
	// TODO_TECHDEBT: Determine the simulation weight value
	defaultWeightMsgCreateClaim int = 100

	opWeightMsgSubmitProof = "op_weight_msg_submit_proof"
	// TODO_TECHDEBT: Determine the simulation weight value
	defaultWeightMsgSubmitProof int = 100

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
	proofGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&proofGenesis)
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

	var weightMsgCreateClaim int
	simState.AppParams.GetOrGenerate(opWeightMsgCreateClaim, &weightMsgCreateClaim, nil,
		func(_ *rand.Rand) {
			weightMsgCreateClaim = defaultWeightMsgCreateClaim
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCreateClaim,
		proofsimulation.SimulateMsgCreateClaim(am.accountKeeper, am.keeper),
	))

	var weightMsgSubmitProof int
	simState.AppParams.GetOrGenerate(opWeightMsgSubmitProof, &weightMsgSubmitProof, nil,
		func(_ *rand.Rand) {
			weightMsgSubmitProof = defaultWeightMsgSubmitProof
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgSubmitProof,
		proofsimulation.SimulateMsgSubmitProof(am.accountKeeper, am.keeper),
	))

	var weightMsgUpdateParam int
	simState.AppParams.GetOrGenerate(opWeightMsgUpdateParam, &weightMsgUpdateParam, nil,
		func(_ *rand.Rand) {
			weightMsgUpdateParam = defaultWeightMsgUpdateParam
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUpdateParam,
		proofsimulation.SimulateMsgUpdateParam(am.accountKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgCreateClaim,
			defaultWeightMsgCreateClaim,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				proofsimulation.SimulateMsgCreateClaim(am.accountKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgSubmitProof,
			defaultWeightMsgSubmitProof,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				proofsimulation.SimulateMsgSubmitProof(am.accountKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgUpdateParam,
			defaultWeightMsgUpdateParam,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				proofsimulation.SimulateMsgUpdateParam(am.accountKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
