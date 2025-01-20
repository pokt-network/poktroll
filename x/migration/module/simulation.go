package migration

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/pokt-network/poktroll/testutil/sample"
	migrationsimulation "github.com/pokt-network/poktroll/x/migration/simulation"
	"github.com/pokt-network/poktroll/x/migration/types"
)

// avoid unused import issue
var (
	_ = migrationsimulation.FindAccount
	_ = rand.Rand{}
	_ = sample.AccAddress
	_ = sdk.AccAddress{}
	_ = simulation.MsgEntryKind
)

const (
	opWeightMsgUploadMorseState = "op_weight_msg_upload_morse_state"
	// TODO: Determine the simulation weight value
	defaultWeightMsgUploadMorseState int = 100

	opWeightMsgClaimMorsePokt = "op_weight_msg_claim_morse_pokt"
	// TODO: Determine the simulation weight value
	defaultWeightMsgClaimMorsePokt int = 100

	// this line is used by starport scaffolding # simapp/module/const
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	migrationGenesis := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # simapp/module/genesisState
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&migrationGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)

	var weightMsgUploadMorseState int
	simState.AppParams.GetOrGenerate(opWeightMsgUploadMorseState, &weightMsgUploadMorseState, nil,
		func(_ *rand.Rand) {
			weightMsgUploadMorseState = defaultWeightMsgUploadMorseState
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgUploadMorseState,
		migrationsimulation.SimulateMsgUploadMorseState(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgClaimMorsePokt int
	simState.AppParams.GetOrGenerate(opWeightMsgClaimMorsePokt, &weightMsgClaimMorsePokt, nil,
		func(_ *rand.Rand) {
			weightMsgClaimMorsePokt = defaultWeightMsgClaimMorsePokt
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgClaimMorsePokt,
		migrationsimulation.SimulateMsgClaimMorsePokt(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgUploadMorseState,
			defaultWeightMsgUploadMorseState,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				migrationsimulation.SimulateMsgUploadMorseState(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgClaimMorsePokt,
			defaultWeightMsgClaimMorsePokt,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				migrationsimulation.SimulateMsgClaimMorsePokt(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
