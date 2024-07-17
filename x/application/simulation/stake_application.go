package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

func SimulateMsgStakeApplication(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		stakeMsg := &application.MsgStakeApplication{
			Address: simAccount.Address.String(),
		}

		// TODO: Handling the StakeApplication simulation
		// See the documentation here to simulate application staking: https://docs.cosmos.network/main/learn/advanced/simulation

		return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(stakeMsg), "StakeApplication simulation not implemented"), nil, nil
	}
}
