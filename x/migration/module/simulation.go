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
	opWeightMsgImportMorseClaimableAccounts = "op_weight_msg_import_morse_claimable_accounts"
	// TODO: Determine the simulation weight value
	defaultWeightMsgImportMorseClaimableAccounts int = 100

	opWeightMsgClaimMorseAccount = "op_weight_msg_claim_morse_account"
	// TODO: Determine the simulation weight value
	defaultWeightMsgClaimMorseAccount int = 100

	opWeightMsgClaimMorseApplication = "op_weight_msg_claim_morse_application"
	// TODO: Determine the simulation weight value
	defaultWeightMsgClaimMorseApplication int = 100

	opWeightMsgClaimMorseSupplier = "op_weight_msg_claim_morse_supplier"
	// TODO: Determine the simulation weight value
	defaultWeightMsgClaimMorseSupplier int = 100

	opWeightMsgClaimMorseGateway = "op_weight_msg_claim_morse_gateway"
	// TODO: Determine the simulation weight value
	defaultWeightMsgClaimMorseGateway int = 100

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

	var weightMsgImportMorseClaimableAccounts int
	simState.AppParams.GetOrGenerate(opWeightMsgImportMorseClaimableAccounts, &weightMsgImportMorseClaimableAccounts, nil,
		func(_ *rand.Rand) {
			weightMsgImportMorseClaimableAccounts = defaultWeightMsgImportMorseClaimableAccounts
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgImportMorseClaimableAccounts,
		migrationsimulation.SimulateMsgImportMorseClaimableAccounts(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgClaimMorseAccount int
	simState.AppParams.GetOrGenerate(opWeightMsgClaimMorseAccount, &weightMsgClaimMorseAccount, nil,
		func(_ *rand.Rand) {
			weightMsgClaimMorseAccount = defaultWeightMsgClaimMorseAccount
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgClaimMorseAccount,
		migrationsimulation.SimulateMsgClaimMorseAccount(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgClaimMorseApplication int
	simState.AppParams.GetOrGenerate(opWeightMsgClaimMorseApplication, &weightMsgClaimMorseApplication, nil,
		func(_ *rand.Rand) {
			weightMsgClaimMorseApplication = defaultWeightMsgClaimMorseApplication
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgClaimMorseApplication,
		migrationsimulation.SimulateMsgClaimMorseApplication(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgClaimMorseSupplier int
	simState.AppParams.GetOrGenerate(opWeightMsgClaimMorseSupplier, &weightMsgClaimMorseSupplier, nil,
		func(_ *rand.Rand) {
			weightMsgClaimMorseSupplier = defaultWeightMsgClaimMorseSupplier
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgClaimMorseSupplier,
		migrationsimulation.SimulateMsgClaimMorseSupplier(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	var weightMsgClaimMorseGateway int
	simState.AppParams.GetOrGenerate(opWeightMsgClaimMorseGateway, &weightMsgClaimMorseGateway, nil,
		func(_ *rand.Rand) {
			weightMsgClaimMorseGateway = defaultWeightMsgClaimMorseGateway
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgClaimMorseGateway,
		migrationsimulation.SimulateMsgClaimMorseGateway(am.accountKeeper, am.bankKeeper, am.keeper),
	))

	// this line is used by starport scaffolding # simapp/module/operation

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			opWeightMsgImportMorseClaimableAccounts,
			defaultWeightMsgImportMorseClaimableAccounts,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				migrationsimulation.SimulateMsgImportMorseClaimableAccounts(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgClaimMorseAccount,
			defaultWeightMsgClaimMorseAccount,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				migrationsimulation.SimulateMsgClaimMorseAccount(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgClaimMorseApplication,
			defaultWeightMsgClaimMorseApplication,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				migrationsimulation.SimulateMsgClaimMorseApplication(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgClaimMorseSupplier,
			defaultWeightMsgClaimMorseSupplier,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				migrationsimulation.SimulateMsgClaimMorseSupplier(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		simulation.NewWeightedProposalMsg(
			opWeightMsgClaimMorseGateway,
			defaultWeightMsgClaimMorseGateway,
			func(r *rand.Rand, ctx sdk.Context, accs []simtypes.Account) sdk.Msg {
				migrationsimulation.SimulateMsgClaimMorseGateway(am.accountKeeper, am.bankKeeper, am.keeper)
				return nil
			},
		),
		// this line is used by starport scaffolding # simapp/module/OpMsg
	}
}
