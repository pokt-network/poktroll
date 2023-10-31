package simulation

import (
	"math/rand"

	"pocket/x/application/keeper"
	"pocket/x/application/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgDelegateToGateway(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAppAccount, _ := simtypes.RandomAcc(r, accs)
		simGatewayAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgDelegateToGateway{
			AppAddress:     simAppAccount.Address.String(),
			GatewayAddress: simGatewayAccount.Address.String(),
		}

		// TODO: Handling the DelegateToGateway simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "DelegateToGateway simulation not implemented"), nil, nil
	}
}
