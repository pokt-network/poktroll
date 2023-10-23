package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"pocket/x/gateway/keeper"
	"pocket/x/gateway/types"
)

func SimulateMsgStakeGateway(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		stakeMsg := &types.MsgStakeGateway{
			Address: simAccount.Address.String(),
		}

		// TODO: Handling the StakeGateway simulation

		return simtypes.NoOpMsg(types.ModuleName, stakeMsg.Type(), "StakeGateway simulation not implemented"), nil, nil
	}
}
