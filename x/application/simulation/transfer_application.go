package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
)

func SimulateMsgTransferApplication(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simSrcAppAccount, _ := simtypes.RandomAcc(r, accs)
		simDstAppAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgTransferApplication{
			SourceAddress:      simSrcAppAccount.Address.String(),
			DestinationAddress: simDstAppAccount.Address.String(),
		}

		// TODO_TECHDEBT: Handling the TransferApplication simulation

		return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "TransferApplication simulation not implemented"), nil, nil
	}
}
