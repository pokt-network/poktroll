package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"pocket/x/supplier/keeper"
	"pocket/x/supplier/types"
)

func SimulateMsgCreateClaim(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgCreateClaim{
			SupplierAddress: simAccount.Address.String(),
		}

		// TODO: Handling the CreateClaim simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "CreateClaim simulation not implemented"), nil, nil
	}
}
